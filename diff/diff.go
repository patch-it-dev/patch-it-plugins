// Package diff compares two OpenAPI specs and reports what structurally
// changed between them.
//
// The one rule that shapes everything here: a $ref is never followed. Every
// schema is compared once, where it is defined, and a property that points at
// another schema is only checked for whether it now points somewhere else.
// Expanding references instead would report a single change once per path that
// can reach it: on Stripe's spec that turns ~60 real changes into ~380,000.
//
// Request and response sides are told apart by location, which is how Stripe
// lays its spec out: v1 request bodies are inline under paths, and response
// objects live in components.schemas. Other APIs may need their own rule.
package diff

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/event"
)

// Kind is the shape of one structural difference. It is deliberately finer
// than event.ChangeType: "property removed" and "enum value removed" are both
// breaking, but they need different sentences and different fingerprints.
type Kind string

const (
	OperationAdded      Kind = "operation_added"
	OperationRemoved    Kind = "operation_removed"
	OperationDeprecated Kind = "operation_deprecated"
	SchemaAdded         Kind = "schema_added"
	SchemaRemoved       Kind = "schema_removed"
	SchemaDeprecated    Kind = "schema_deprecated"
	PropertyAdded       Kind = "property_added"
	PropertyRemoved     Kind = "property_removed"
	PropertyDeprecated  Kind = "property_deprecated"
	BecameRequired      Kind = "became_required"
	BecameOptional      Kind = "became_optional"
	EnumValueAdded      Kind = "enum_value_added"
	EnumValueRemoved    Kind = "enum_value_removed"
	MemberAdded         Kind = "member_added"
	MemberRemoved       Kind = "member_removed"
	TypeChanged         Kind = "type_changed"
)

// Change is one structural difference, located precisely enough that the
// patch-it client can act on it and a human can check it against the spec.
type Change struct {
	Kind       Kind
	ChangeType event.ChangeType
	Breaking   bool
	Side       event.Side

	Operation string // "POST /v1/charges", on the request side
	Schema    string // component schema name, on the response side
	Property  string // dotted path inside the operation or schema; [] marks array items

	Detail  string // what distinguishes this change at this location, e.g. the enum value
	Pointer string // RFC 6901 pointer to the evidence
	InOld   bool   // the evidence lives in the old spec (true for removals)
	Node    any    // the spec node the pointer addresses
}

// Spec is a parsed OpenAPI document.
type Spec struct {
	Version string
	doc     map[string]any
}

// Parse reads a spec. The document is kept as plain JSON rather than typed
// OpenAPI structs: the differ walks it generically, and vendor extensions and
// unknown keywords survive the round trip.
func Parse(data []byte) (*Spec, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing spec: %w", err)
	}
	version, _ := obj(doc["info"])["version"].(string)
	return &Spec{Version: version, doc: doc}, nil
}

// methods are the operation keys a path item can hold. Anything else under a
// path (parameters, servers, summary) is not an operation.
var methods = []string{"get", "put", "post", "delete", "patch", "head", "options", "trace"}

// Options adapts the differ to a spec whose conventions aren't Stripe's. The
// zero value is what Changes does.
type Options struct {
	// PathTemplates matches an operation across the two specs by its path
	// with the parameter names left out, so GET /drives/{name} and
	// GET /drives/{nameOrId} are one operation: a caller sends the same URL to
	// both. Path parameters are then paired by their place in the path, not
	// their name.
	//
	// The trade: a URL that keeps its shape but changes meaning
	// (/teams/{teamId} becoming /teams/{slug}) reads as no change. And when
	// one spec has two paths with the same template, they're paired by exact
	// path instead, so a rename there still reads as a removal.
	PathTemplates bool

	// DeprecatedSummary reports whether an operation's summary marks it
	// deprecated, for a spec that often says so in words and not in the
	// `deprecated` flag. Nil reads only the flag.
	DeprecatedSummary func(summary string) bool

	// DeprecatedDescription reports whether an operation's description marks
	// it deprecated, for a spec that says so in the description's words
	// ("## Deprecated", "This endpoint is DEPRECATED.") and seldom in the
	// flag. Nil doesn't read the description. A deprecation said only there
	// cites the description.
	DeprecatedDescription func(description string) bool

	// QuoteDeprecatedOperation makes a deprecation by the flag cite the whole
	// operation, not the bare `true`: for a spec that says why in the
	// operation's description ("This endpoint is deprecated - use the deploy
	// endpoint"). The excerpt is the operation serialized with sorted keys, so
	// `deprecated` comes before `description` and survives the 2,000-character
	// cut (only a `callbacks` key would sort ahead of it).
	QuoteDeprecatedOperation bool
}

// OperationDeprecated reports whether an operation object is marked
// deprecated, by its flag or, when DeprecatedSummary or DeprecatedDescription
// is set, its summary or description.
func (o Options) OperationDeprecated(op map[string]any) bool {
	return truthy(op["deprecated"]) || o.summaryDeprecated(op) || o.descriptionDeprecated(op)
}

func (o Options) descriptionDeprecated(op map[string]any) bool {
	description, _ := op["description"].(string)
	return o.DeprecatedDescription != nil && description != "" && o.DeprecatedDescription(description)
}

func (o Options) summaryDeprecated(op map[string]any) bool {
	summary, _ := op["summary"].(string)
	return o.DeprecatedSummary != nil && summary != "" && o.DeprecatedSummary(summary)
}

// Changes reports what changed between two specs, oldest first. Either spec
// being nil means there is nothing to compare, which is the first-run case.
// Identical specs produce no changes.
func Changes(oldSpec, newSpec *Spec) []Change {
	return ChangesWith(oldSpec, newSpec, Options{})
}

// ChangesWith is Changes, adapted by opts.
func ChangesWith(oldSpec, newSpec *Spec, opts Options) []Change {
	if oldSpec == nil || newSpec == nil {
		return nil
	}
	d := &differ{opts: opts}
	d.paths(obj(oldSpec.doc["paths"]), obj(newSpec.doc["paths"]))
	d.schemas(
		obj(obj(oldSpec.doc["components"])["schemas"]),
		obj(obj(newSpec.doc["components"])["schemas"]),
	)
	sort.SliceStable(d.changes, func(i, j int) bool {
		a, b := d.changes[i], d.changes[j]
		if a.Pointer != b.Pointer {
			return a.Pointer < b.Pointer
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Detail < b.Detail
	})
	return d.changes
}

type differ struct {
	opts    Options
	changes []Change
}

// where carries the location of whatever is currently being compared, so the
// schema walk can be shared between request bodies and response schemas.
type where struct {
	side      event.Side
	operation string
	schema    string
	property  string // dotted path so far, "" at the top of a schema
	ptrOld    string
	ptrNew    string
}

func (w where) prop(name string) where {
	w.property = joinProp(w.property, name)
	w.ptrOld += "/properties/" + escape(name)
	w.ptrNew += "/properties/" + escape(name)
	return w
}

func (w where) items() where {
	w.property += "[]"
	w.ptrOld += "/items"
	w.ptrNew += "/items"
	return w
}

func (w where) member(keyword string, i int) where {
	at := fmt.Sprintf("/%s/%d", keyword, i)
	w.ptrOld += at
	w.ptrNew += at
	return w
}

func (w where) child(segment string) where {
	w.ptrOld += "/" + segment
	w.ptrNew += "/" + segment
	return w
}

// add records a change, dropping the ones that affect nobody. Silence beats
// noise: a response property that becomes required, or a request field that
// becomes optional, cannot break a caller, so no event is produced.
func (d *differ) add(w where, kind Kind, detail string, inOld bool, node any) {
	changeType, breaking, emit := classify(kind, w.side)
	if !emit {
		return
	}
	ptr := w.ptrNew
	if inOld {
		ptr = w.ptrOld
	}
	d.changes = append(d.changes, Change{
		Kind:       kind,
		ChangeType: changeType,
		Breaking:   breaking,
		Side:       w.side,
		Operation:  w.operation,
		Schema:     w.schema,
		Property:   w.property,
		Detail:     detail,
		Pointer:    ptr,
		InOld:      inOld,
		Node:       node,
	})
}

// classify turns a kind plus a side into a change type. It is the table in
// README.md, and the only place breaking-ness is decided.
func classify(kind Kind, side event.Side) (event.ChangeType, bool, bool) {
	request := side == event.SideRequest
	switch kind {
	case OperationAdded, SchemaAdded:
		return event.ChangeTypeNewFeature, false, true
	case OperationRemoved, SchemaRemoved, PropertyRemoved, TypeChanged:
		return event.ChangeTypeBreaking, true, true
	case OperationDeprecated, SchemaDeprecated, PropertyDeprecated:
		return event.ChangeTypeDeprecation, false, true
	case PropertyAdded, EnumValueAdded, MemberAdded:
		// Adding to a request is only breaking when the new field is required,
		// and that case is emitted as BecameRequired instead of PropertyAdded.
		return event.ChangeTypeNewFeature, false, true
	case EnumValueRemoved, MemberRemoved:
		// Callers can no longer send the value; readers never had to handle it.
		if request {
			return event.ChangeTypeBreaking, true, true
		}
		return "", false, false
	case BecameRequired:
		// Senders must now provide it. A response gaining a guarantee is fine.
		if request {
			return event.ChangeTypeBreaking, true, true
		}
		return "", false, false
	case BecameOptional:
		// A response field that may now be missing breaks readers.
		if request {
			return "", false, false
		}
		return event.ChangeTypeBreaking, true, true
	default:
		return "", false, false
	}
}

// ---------- request side: paths ----------

func (d *differ) paths(oldPaths, newPaths map[string]any) {
	for _, pair := range d.operations(oldPaths, newPaths) {
		method := pair.method
		oldOp := obj(obj(oldPaths[pair.oldPath])[method])
		newOp := obj(obj(newPaths[pair.newPath])[method])
		path := pair.newPath
		if newOp == nil {
			path = pair.oldPath
		}
		w := where{
			side:      event.SideRequest,
			operation: strings.ToUpper(method) + " " + path,
			ptrOld:    "/paths/" + escape(pair.oldPath) + "/" + method,
			ptrNew:    "/paths/" + escape(pair.newPath) + "/" + method,
		}
		switch {
		case oldOp == nil:
			d.add(w, OperationAdded, "", false, newOp)
		case newOp == nil:
			d.add(w, OperationRemoved, "", true, oldOp)
		default:
			if !d.opts.OperationDeprecated(oldOp) && d.opts.OperationDeprecated(newOp) {
				switch {
				case truthy(newOp["deprecated"]) && d.opts.QuoteDeprecatedOperation:
					d.add(w, OperationDeprecated, "", false, newOp)
				case truthy(newOp["deprecated"]):
					d.add(w.child("deprecated"), OperationDeprecated, "", false, newOp["deprecated"])
				case d.opts.summaryDeprecated(newOp):
					// Said in words: the summary is the evidence.
					d.add(w.child("summary"), OperationDeprecated, "", false, newOp["summary"])
				default:
					// Said in the description's words: it is the evidence.
					d.add(w.child("description"), OperationDeprecated, "", false, newOp["description"])
				}
			}
			d.parameters(w, oldOp, newOp, pathParamRenames(pair.oldPath, pair.newPath))
			d.requestBody(w, oldOp, newOp)
		}
	}
}

// opPair is one operation as each spec has it. The path is empty on the side
// that lacks it; with PathTemplates the two paths may differ in their
// parameter names.
type opPair struct {
	method, oldPath, newPath string
}

// operations pairs every operation in either spec with its counterpart in the
// other, by method and path, or, with PathTemplates, by method and template.
func (d *differ) operations(oldPaths, newPaths map[string]any) []opPair {
	key := func(method, path string) string {
		if d.opts.PathTemplates {
			path = pathParam.ReplaceAllString(path, "{}")
		}
		return method + " " + path
	}
	index := func(paths map[string]any) map[string][]string {
		ix := map[string][]string{}
		for path, item := range paths {
			for _, method := range methods {
				if obj(obj(item)[method]) != nil {
					k := key(method, path)
					ix[k] = append(ix[k], path)
				}
			}
		}
		return ix
	}
	oldIx, newIx := index(oldPaths), index(newPaths)

	var pairs []opPair
	for _, k := range union(oldIx, newIx) {
		method, _, _ := strings.Cut(k, " ")
		oldList, newList := oldIx[k], newIx[k]
		if len(oldList) <= 1 && len(newList) <= 1 {
			p := opPair{method: method}
			if len(oldList) == 1 {
				p.oldPath = oldList[0]
			}
			if len(newList) == 1 {
				p.newPath = newList[0]
			}
			pairs = append(pairs, p)
			continue
		}
		// Two paths in one spec share the template, so it can't say which is
		// which: pair them by exact path, as without PathTemplates.
		all := map[string]struct{}{}
		for _, p := range append(append([]string{}, oldList...), newList...) {
			all[p] = struct{}{}
		}
		for _, path := range sortedKeys(all) {
			p := opPair{method: method}
			if _, ok := oldPaths[path]; ok && obj(obj(oldPaths[path])[method]) != nil {
				p.oldPath = path
			}
			if _, ok := newPaths[path]; ok && obj(obj(newPaths[path])[method]) != nil {
				p.newPath = path
			}
			pairs = append(pairs, p)
		}
	}
	return pairs
}

// pathParam is one {parameter} in a path template.
var pathParam = regexp.MustCompile(`\{[^}]*\}`)

// pathParamRenames maps each path parameter's name in oldPath to its name in
// newPath, by place, when the two paths are one template; nil otherwise.
func pathParamRenames(oldPath, newPath string) map[string]string {
	if oldPath == newPath || oldPath == "" || newPath == "" {
		return nil
	}
	oldNames, newNames := pathParam.FindAllString(oldPath, -1), pathParam.FindAllString(newPath, -1)
	if len(oldNames) != len(newNames) {
		return nil
	}
	renames := map[string]string{}
	for i := range oldNames {
		renames[strings.Trim(oldNames[i], "{}")] = strings.Trim(newNames[i], "{}")
	}
	return renames
}

// parameters compares an operation's query and path parameters by name. Their
// index in the array is not stable between versions, so it is only used to
// point at the evidence. renames gives the new name of each old path parameter
// when the path's parameter names changed.
func (d *differ) parameters(w where, oldOp, newOp map[string]any, renames map[string]string) {
	oldParams := byName(oldOp["parameters"], renames)
	oldPtr := indexByName(oldOp["parameters"], renames)
	newParams := byName(newOp["parameters"], nil)
	newPtr := indexByName(newOp["parameters"], nil)

	for _, name := range union(oldParams, newParams) {
		oldParam, newParam := obj(oldParams[name]), obj(newParams[name])
		pw := w
		pw.property = name
		pw.ptrOld = fmt.Sprintf("%s/parameters/%d", w.ptrOld, oldPtr[name])
		pw.ptrNew = fmt.Sprintf("%s/parameters/%d", w.ptrNew, newPtr[name])

		switch {
		case oldParam == nil:
			// One change, one event: a new required parameter is breaking for
			// every caller, so it is reported only as that.
			if truthy(newParam["required"]) {
				d.add(pw, BecameRequired, "added as required", false, newParam)
			} else {
				d.add(pw, PropertyAdded, "", false, newParam)
			}
		case newParam == nil:
			d.add(pw, PropertyRemoved, "", true, oldParam)
		default:
			switch {
			case !truthy(oldParam["required"]) && truthy(newParam["required"]):
				d.add(pw, BecameRequired, "", false, newParam)
			case truthy(oldParam["required"]) && !truthy(newParam["required"]):
				d.add(pw, BecameOptional, "", false, newParam)
			}
			d.schema(pw.child("schema"), obj(oldParam["schema"]), obj(newParam["schema"]))
		}
	}
}

// requestBody compares the inline schema of each media type. A body that is a
// $ref is not followed, by the rule at the top of this file.
func (d *differ) requestBody(w where, oldOp, newOp map[string]any) {
	oldContent := obj(obj(oldOp["requestBody"])["content"])
	newContent := obj(obj(newOp["requestBody"])["content"])

	for _, mediaType := range union(oldContent, newContent) {
		oldSchema := obj(obj(oldContent[mediaType])["schema"])
		newSchema := obj(obj(newContent[mediaType])["schema"])
		if oldSchema == nil || newSchema == nil {
			continue // a body that appears or disappears rides on the operation
		}
		bw := w.child("requestBody").child("content").child(escape(mediaType)).child("schema")
		d.schema(bw, oldSchema, newSchema)
	}
}

// ---------- response side: components.schemas ----------

func (d *differ) schemas(oldSchemas, newSchemas map[string]any) {
	for _, name := range union(oldSchemas, newSchemas) {
		oldSchema, newSchema := obj(oldSchemas[name]), obj(newSchemas[name])
		w := where{
			side:   event.SideResponse,
			schema: name,
			ptrOld: "/components/schemas/" + escape(name),
			ptrNew: "/components/schemas/" + escape(name),
		}
		switch {
		case oldSchema == nil:
			d.add(w, SchemaAdded, "", false, newSchema)
		case newSchema == nil:
			d.add(w, SchemaRemoved, "", true, oldSchema)
		default:
			d.schema(w, oldSchema, newSchema)
		}
	}
}

// ---------- shared schema walk ----------

// schema compares two schema objects that sit at the same place in both specs.
// It recurses through inline sub-schemas only; a $ref is a boundary.
func (d *differ) schema(w where, oldSchema, newSchema map[string]any) {
	if oldSchema == nil || newSchema == nil {
		return
	}

	oldRef, _ := oldSchema["$ref"].(string)
	newRef, _ := newSchema["$ref"].(string)
	if oldRef != "" || newRef != "" {
		if oldRef != newRef {
			d.add(w, TypeChanged, describeRefChange(oldRef, newRef), false, newSchema)
		}
		return // never follow a reference
	}

	if !truthy(oldSchema["deprecated"]) && truthy(newSchema["deprecated"]) {
		kind := PropertyDeprecated
		if w.property == "" {
			kind = SchemaDeprecated
		}
		d.add(w.child("deprecated"), kind, "", false, newSchema["deprecated"])
	}

	if oldType, newType := typeOf(oldSchema), typeOf(newSchema); oldType != newType && oldType != "" && newType != "" {
		d.add(w, TypeChanged, oldType+" → "+newType, false, newSchema)
	}

	d.enums(w, oldSchema, newSchema)
	d.properties(w, oldSchema, newSchema)

	if oldItems, newItems := obj(oldSchema["items"]), obj(newSchema["items"]); oldItems != nil && newItems != nil {
		d.schema(w.items(), oldItems, newItems)
	}

	for _, keyword := range []string{"anyOf", "oneOf", "allOf"} {
		d.composed(w, keyword, arr(oldSchema[keyword]), arr(newSchema[keyword]))
	}
}

func (d *differ) enums(w where, oldSchema, newSchema map[string]any) {
	oldValues, newValues := enumSet(oldSchema["enum"]), enumSet(newSchema["enum"])
	if oldValues == nil || newValues == nil {
		return
	}
	ew := w.child("enum")
	for _, v := range sortedKeys(newValues) {
		if _, ok := oldValues[v]; !ok {
			d.add(ew, EnumValueAdded, v, false, newSchema["enum"])
		}
	}
	for _, v := range sortedKeys(oldValues) {
		if _, ok := newValues[v]; !ok {
			d.add(ew, EnumValueRemoved, v, true, oldSchema["enum"])
		}
	}
}

func (d *differ) properties(w where, oldSchema, newSchema map[string]any) {
	oldProps, newProps := obj(oldSchema["properties"]), obj(newSchema["properties"])
	oldRequired, newRequired := stringSet(oldSchema["required"]), stringSet(newSchema["required"])

	for _, name := range union(oldProps, newProps) {
		oldProp, newProp := obj(oldProps[name]), obj(newProps[name])
		pw := w.prop(name)
		_, wasRequired := oldRequired[name]
		_, isRequired := newRequired[name]

		switch {
		case oldProp == nil:
			// One change, one event. On the request side a new required field
			// is breaking and reported only as that; emitting a new_feature
			// alongside it would make the same field look harmless. On the
			// response side a required field is still just an addition
			// (classify drops BecameRequired there), so it stays PropertyAdded.
			if isRequired && w.side == event.SideRequest {
				d.add(pw, BecameRequired, "added as required", false, newProp)
			} else {
				d.add(pw, PropertyAdded, "", false, newProp)
			}
		case newProp == nil:
			d.add(pw, PropertyRemoved, "", true, oldProp)
		default:
			switch {
			case !wasRequired && isRequired:
				d.add(pw, BecameRequired, "", false, newProp)
			case wasRequired && !isRequired:
				d.add(pw, BecameOptional, "", false, newProp)
			}
			d.schema(pw, oldProp, newProp)
		}
	}
}

// composed compares anyOf/oneOf/allOf. Inline members are matched by position,
// because that is the only identity they have; $ref members are matched as a
// set of schema names, so reordering them is not a change.
func (d *differ) composed(w where, keyword string, oldMembers, newMembers []any) {
	oldRefs, newRefs := refSet(oldMembers), refSet(newMembers)
	kw := w.child(keyword)
	for _, ref := range sortedKeys(newRefs) {
		if _, ok := oldRefs[ref]; !ok {
			d.add(kw, MemberAdded, ref, false, newMembers)
		}
	}
	for _, ref := range sortedKeys(oldRefs) {
		if _, ok := newRefs[ref]; !ok {
			d.add(kw, MemberRemoved, ref, true, oldMembers)
		}
	}

	for i := 0; i < len(oldMembers) && i < len(newMembers); i++ {
		oldMember, newMember := obj(oldMembers[i]), obj(newMembers[i])
		if oldMember == nil || newMember == nil {
			continue
		}
		if oldMember["$ref"] != nil || newMember["$ref"] != nil {
			continue // handled as a set above
		}
		d.schema(w.member(keyword, i), oldMember, newMember)
	}
}

// ---------- helpers ----------

func obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func arr(v any) []any {
	a, _ := v.([]any)
	return a
}

func truthy(v any) bool {
	b, _ := v.(bool)
	return b
}

// typeOf reports a schema's declared type. OpenAPI 3.1 allows a list of types,
// which is flattened so a change in the set still reads as a type change.
func typeOf(schema map[string]any) string {
	switch t := schema["type"].(type) {
	case string:
		return t
	case []any:
		parts := make([]string, 0, len(t))
		for _, v := range t {
			parts = append(parts, fmt.Sprint(v))
		}
		sort.Strings(parts)
		return strings.Join(parts, "|")
	default:
		return ""
	}
}

func describeRefChange(oldRef, newRef string) string {
	return refName(oldRef) + " → " + refName(newRef)
}

func refName(ref string) string {
	if ref == "" {
		return "(none)"
	}
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

func refSet(members []any) map[string]struct{} {
	refs := map[string]struct{}{}
	for _, m := range members {
		if ref, ok := obj(m)["$ref"].(string); ok && ref != "" {
			refs[refName(ref)] = struct{}{}
		}
	}
	return refs
}

func enumSet(v any) map[string]struct{} {
	values := arr(v)
	if values == nil {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[fmt.Sprint(value)] = struct{}{}
	}
	return set
}

func stringSet(v any) map[string]struct{} {
	set := map[string]struct{}{}
	for _, item := range arr(v) {
		if s, ok := item.(string); ok {
			set[s] = struct{}{}
		}
	}
	return set
}

// byName keys parameters by their name; indexByName remembers where each one
// sat, so the pointer can address the right array element in each spec. A path
// parameter named in renames is keyed by its new name.
func byName(v any, renames map[string]string) map[string]any {
	params := map[string]any{}
	for _, item := range arr(v) {
		if name, ok := paramName(item, renames); ok {
			params[name] = item
		}
	}
	return params
}

func indexByName(v any, renames map[string]string) map[string]int {
	indexes := map[string]int{}
	for i, item := range arr(v) {
		if name, ok := paramName(item, renames); ok {
			indexes[name] = i
		}
	}
	return indexes
}

func paramName(item any, renames map[string]string) (string, bool) {
	p := obj(item)
	name, ok := p["name"].(string)
	if !ok {
		return "", false
	}
	if to, renamed := renames[name]; renamed && p["in"] == "path" {
		return to, true
	}
	return name, true
}

// union lists every key in either map, sorted, so output order never depends
// on Go's map iteration.
func union[V any](a, b map[string]V) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}
	return sortedKeys(seen)
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func joinProp(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// escape encodes a JSON Pointer token per RFC 6901.
func escape(token string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(token)
}
