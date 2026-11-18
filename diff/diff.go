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

// Changes reports what changed between two specs, oldest first. Either spec
// being nil means there is nothing to compare, which is the first-run case.
// Identical specs produce no changes.
func Changes(oldSpec, newSpec *Spec) []Change {
	if oldSpec == nil || newSpec == nil {
		return nil
	}
	d := &differ{}
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
	for _, path := range union(oldPaths, newPaths) {
		oldItem, newItem := obj(oldPaths[path]), obj(newPaths[path])
		ptrOld := "/paths/" + escape(path)
		ptrNew := ptrOld

		for _, method := range methods {
			oldOp, newOp := obj(oldItem[method]), obj(newItem[method])
			if oldOp == nil && newOp == nil {
				continue
			}
			w := where{
				side:      event.SideRequest,
				operation: strings.ToUpper(method) + " " + path,
				ptrOld:    ptrOld + "/" + method,
				ptrNew:    ptrNew + "/" + method,
			}
			switch {
			case oldOp == nil:
				d.add(w, OperationAdded, "", false, newOp)
			case newOp == nil:
				d.add(w, OperationRemoved, "", true, oldOp)
			default:
				if !truthy(oldOp["deprecated"]) && truthy(newOp["deprecated"]) {
					d.add(w.child("deprecated"), OperationDeprecated, "", false, newOp["deprecated"])
				}
				d.parameters(w, oldOp, newOp)
				d.requestBody(w, oldOp, newOp)
			}
		}
	}
}

// parameters compares an operation's query and path parameters by name. Their
// index in the array is not stable between versions, so it is only used to
// point at the evidence.
func (d *differ) parameters(w where, oldOp, newOp map[string]any) {
	oldParams := byName(oldOp["parameters"])
	oldPtr := indexByName(oldOp["parameters"])
	newParams := byName(newOp["parameters"])
	newPtr := indexByName(newOp["parameters"])

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
// sat, so the pointer can address the right array element in each spec.
func byName(v any) map[string]any {
	params := map[string]any{}
	for _, item := range arr(v) {
		if name, ok := obj(item)["name"].(string); ok {
			params[name] = item
		}
	}
	return params
}

func indexByName(v any) map[string]int {
	indexes := map[string]int{}
	for i, item := range arr(v) {
		if name, ok := obj(item)["name"].(string); ok {
			indexes[name] = i
		}
	}
	return indexes
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
