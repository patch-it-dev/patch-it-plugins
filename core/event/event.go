// Package event is the change event of patch-it's contract
// (docs/contract/change-event.md in patch-it): what changed, at which
// version, with the source that says so. A typed subject says what changed.
package event

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// ChangeType is what kind of change it is to the code that calls the source.
type ChangeType string

const (
	ChangeTypeBreaking    ChangeType = "breaking"
	ChangeTypeDeprecation ChangeType = "deprecation"
	ChangeTypeNewFeature  ChangeType = "new_feature"
	ChangeTypeFix         ChangeType = "fix"
)

// Kind is what an api_operation subject is about.
type Kind string

const (
	KindOperation        Kind = "operation"
	KindRequestParameter Kind = "request_parameter"
	KindSchema           Kind = "schema"
	KindSchemaProperty   Kind = "schema_property"
)

// Side is which side of a call an api_operation change breaks.
type Side string

const (
	SideRequest  Side = "request"
	SideResponse Side = "response"
)

// SubjectType is what kind of thing changed.
type SubjectType string

const (
	SubjectAPIOperation   SubjectType = "api_operation"
	SubjectModel          SubjectType = "model"
	SubjectPackageRelease SubjectType = "package_release"
	SubjectCIRuntime      SubjectType = "ci_runtime"
)

// PackageChange is what a package release did to the code that calls it.
type PackageChange string

const (
	PackageRemovedParameter PackageChange = "removed_parameter"
	PackageRemovedSymbol    PackageChange = "removed_symbol"
	PackageRenamedSymbol    PackageChange = "renamed_symbol"
	PackageChangedDefault   PackageChange = "changed_default"
)

// Subject is what changed. Type picks which fields it may have; every other
// field must be empty (the contract's "a field its type doesn't define").
type Subject struct {
	Type SubjectType `json:"type"`

	// api_operation
	Kind      Kind   `json:"kind,omitempty"`
	Side      Side   `json:"side,omitempty"`
	Operation string `json:"operation,omitempty"`
	Schema    string `json:"schema,omitempty"`
	Property  string `json:"property,omitempty"`

	// model
	Provider string `json:"provider,omitempty"`
	Name     string `json:"name,omitempty"`

	// package_release
	Ecosystem string        `json:"ecosystem,omitempty"`
	Package   string        `json:"package,omitempty"`
	Change    PackageChange `json:"change,omitempty"`
	Symbols   []string      `json:"symbols,omitempty"`
	Parameter string        `json:"parameter,omitempty"`

	// ci_runtime
	Platform string `json:"platform,omitempty"`
	Runtime  string `json:"runtime,omitempty"`

	// model, package_release (renamed_symbol), ci_runtime
	Replacement string `json:"replacement,omitempty"`
}

// Event is one change event.
type Event struct {
	ID              int64      `json:"id"`
	Fingerprint     string     `json:"fingerprint"`
	API             string     `json:"api"`
	Version         string     `json:"version"`
	PreviousVersion string     `json:"previous_version,omitempty"`
	ChangeType      ChangeType `json:"change_type"`
	Breaking        bool       `json:"breaking"`
	Subject         Subject    `json:"subject"`
	Supersedes      string     `json:"supersedes,omitempty"`
	Description     string     `json:"description"`
	MigrationHint   string     `json:"migration_hint,omitempty"`
	SourceURL       string     `json:"source_url"`
	SourcePointer   string     `json:"source_pointer,omitempty"`
	SourceExcerpt   string     `json:"source_excerpt"`
	Confidence      float64    `json:"confidence"`
	DetectedAt      time.Time  `json:"detected_at"`
}

// Decode reads one event and refuses a field the contract doesn't define,
// at the top level or in subject: the patch-it client's model forbids extra
// fields too, so the hosted service and the client reject the same events. Keys are
// matched exactly ("TYPE" isn't "type", though encoding/json would take it),
// and a subject key belongs to its type even when its value is empty: pydantic
// sees the key, not the value.
func Decode(raw []byte) (Event, error) {
	var e Event
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		return Event{}, err
	}
	var keys struct {
		Top     map[string]json.RawMessage
		Subject map[string]json.RawMessage `json:"subject"`
	}
	if err := json.Unmarshal(raw, &keys.Top); err != nil {
		return Event{}, err
	}
	for k := range keys.Top {
		if !slices.Contains(eventKeys, k) {
			return Event{}, fmt.Errorf("%q isn't a field of a change event", k)
		}
	}
	if sub, ok := keys.Top["subject"]; ok {
		if err := json.Unmarshal(sub, &keys.Subject); err != nil {
			return Event{}, err
		}
	}
	allowed := slices.Concat(subjectKeys[e.Subject.Type], []string{"type"})
	for k := range keys.Subject {
		if !slices.Contains(allowed, k) {
			return Event{}, fmt.Errorf("subject.%s isn't a field of type %s", k, e.Subject.Type)
		}
	}
	return e, nil
}

// eventKeys are the JSON names of Event's fields.
var eventKeys = []string{"id", "fingerprint", "api", "version", "previous_version", "change_type",
	"breaking", "subject", "supersedes", "description", "migration_hint", "source_url",
	"source_pointer", "source_excerpt", "confidence", "detected_at"}

// subjectKeys are the fields each type may have, besides `type`.
var subjectKeys = map[SubjectType][]string{
	SubjectAPIOperation:   {"kind", "side", "operation", "schema", "property"},
	SubjectModel:          {"provider", "name", "replacement"},
	SubjectPackageRelease: {"ecosystem", "package", "change", "symbols", "parameter", "replacement"},
	SubjectCIRuntime:      {"platform", "runtime", "replacement"},
}

// Fingerprint identifies one change, so re-reading the same source stores
// nothing twice. The parts are joined with a separator that can't appear in
// them. For api_operation, the location and detail (what separates two
// changes at one place: the enum value added, "added as required"); for the
// other types, the type and its key fields, detail empty: the key already
// says what changed. Symbols are sorted first, so the same release always
// gets the same fingerprint.
func Fingerprint(api, version string, changeType ChangeType, s Subject, detail string) string {
	if s.Type == SubjectAPIOperation {
		return hash(api, version, string(changeType), string(s.Kind), string(s.Side), s.Operation, s.Schema, s.Property, detail)
	}
	parts := []string{api, version, string(changeType), string(s.Type)}
	switch s.Type {
	case SubjectModel:
		parts = append(parts, s.Provider, s.Name)
	case SubjectPackageRelease:
		symbols := slices.Clone(s.Symbols)
		slices.Sort(symbols)
		parts = append(parts, s.Ecosystem, s.Package, string(s.Change), strings.Join(symbols, ","), s.Parameter)
	case SubjectCIRuntime:
		parts = append(parts, s.Platform, s.Runtime)
	}
	return hash(append(parts, "")...)
}

func hash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Validate applies the contract: the rules for every field, then subject's own.
func (e Event) Validate() error {
	if err := checkCommon(e.SourceURL, e.SourceExcerpt, e.Fingerprint, e.API, e.Version,
		e.Description, e.DetectedAt, e.ChangeType, e.Breaking, e.Confidence); err != nil {
		return err
	}
	if e.Supersedes != "" && !fingerprintPattern.MatchString(e.Supersedes) {
		return fmt.Errorf("supersedes must look like sha256:<64 lowercase hex>, got %q", e.Supersedes)
	}
	return e.Subject.validate()
}

func (s Subject) validate() error {
	allowed, err := s.required()
	if err != nil {
		return err
	}
	for name, set := range s.present() {
		if set && !slices.Contains(allowed, name) {
			return fmt.Errorf("subject.%s isn't a field of type %s", name, s.Type)
		}
	}
	return nil
}

// required checks the fields the type must have and returns every field it may.
func (s Subject) required() ([]string, error) {
	need := func(name, v string) error {
		if v == "" {
			return fmt.Errorf("subject.%s is required for type %s", name, s.Type)
		}
		return nil
	}
	switch s.Type {
	case SubjectAPIOperation:
		if err := s.validateOperation(); err != nil {
			return nil, fmt.Errorf("subject: %w", err)
		}
		return subjectKeys[SubjectAPIOperation], nil
	case SubjectModel:
		for _, f := range [][2]string{{"provider", s.Provider}, {"name", s.Name}} {
			if err := need(f[0], f[1]); err != nil {
				return nil, err
			}
		}
		return subjectKeys[SubjectModel], nil
	case SubjectPackageRelease:
		for _, f := range [][2]string{{"ecosystem", s.Ecosystem}, {"package", s.Package}} {
			if err := need(f[0], f[1]); err != nil {
				return nil, err
			}
		}
		switch s.Change {
		case PackageRemovedParameter, PackageChangedDefault:
			if err := need("parameter", s.Parameter); err != nil {
				return nil, err
			}
		case PackageRenamedSymbol:
			if err := need("replacement", s.Replacement); err != nil {
				return nil, err
			}
		case PackageRemovedSymbol:
		default:
			return nil, fmt.Errorf("subject.change must be a valid enum value, got %q", s.Change)
		}
		if len(s.Symbols) == 0 || slices.Contains(s.Symbols, "") {
			return nil, fmt.Errorf("subject.symbols must name at least one symbol, none empty")
		}
		return subjectKeys[SubjectPackageRelease], nil
	case SubjectCIRuntime:
		for _, f := range [][2]string{{"platform", s.Platform}, {"runtime", s.Runtime}} {
			if err := need(f[0], f[1]); err != nil {
				return nil, err
			}
		}
		return subjectKeys[SubjectCIRuntime], nil
	}
	return nil, fmt.Errorf("subject.type must be a valid enum value, got %q", s.Type)
}

// present is which subject fields are set, by their JSON names.
func (s Subject) present() map[string]bool {
	return map[string]bool{
		"kind": s.Kind != "", "side": s.Side != "", "operation": s.Operation != "",
		"schema": s.Schema != "", "property": s.Property != "",
		"provider": s.Provider != "", "name": s.Name != "",
		"ecosystem": s.Ecosystem != "", "package": s.Package != "", "change": s.Change != "",
		"symbols": len(s.Symbols) > 0, "parameter": s.Parameter != "",
		"platform": s.Platform != "", "runtime": s.Runtime != "",
		"replacement": s.Replacement != "",
	}
}

// fingerprintPattern is the contract's shape for a fingerprint (the schema's
// pattern in the change event contract). patch-it checks the same pattern.
var fingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// checkCommon is the contract's rules for every field outside subject.
func checkCommon(sourceURL, excerpt, fingerprint, api, version, description string,
	detectedAt time.Time, changeType ChangeType, breaking bool, confidence float64) error {
	// Rule 1: provenance is mandatory — the anti-hallucination mechanism.
	if sourceURL == "" {
		return fmt.Errorf("source_url is required")
	}
	if excerpt == "" {
		return fmt.Errorf("source_excerpt is required")
	}

	// Rule 2: other required fields must be non-empty.
	if fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}
	if !fingerprintPattern.MatchString(fingerprint) {
		return fmt.Errorf("fingerprint must look like sha256:<64 lowercase hex>, got %q", fingerprint)
	}
	if api == "" {
		return fmt.Errorf("api is required")
	}
	if version == "" {
		return fmt.Errorf("version is required")
	}
	if description == "" {
		return fmt.Errorf("description is required")
	}
	if detectedAt.IsZero() {
		return fmt.Errorf("detected_at is required")
	}

	// Rule 3: change_type must be one of the known enum values.
	if changeType != ChangeTypeBreaking &&
		changeType != ChangeTypeDeprecation &&
		changeType != ChangeTypeNewFeature &&
		changeType != ChangeTypeFix {
		return fmt.Errorf("change_type must be a valid enum value, got %q", changeType)
	}

	// Rule 4: breaking must agree with change_type.
	if breaking != (changeType == ChangeTypeBreaking) {
		return fmt.Errorf("breaking (%v) must equal change_type == breaking", breaking)
	}

	// Rule 6: confidence must be in [0, 1].
	if confidence < 0 || confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %v", confidence)
	}

	// Rule 7: source_excerpt is capped at 2000 characters. Count runes, not
	// bytes: spec text is full of non-ASCII (Stripe writes "invoice\u2019s"), and
	// len() would reject excerpts well under the documented limit.
	if n := utf8.RuneCountInString(excerpt); n > 2000 {
		return fmt.Errorf("source_excerpt must be at most 2000 characters, got %d", n)
	}
	return nil
}

// validateOperation is the contract's rules for an api_operation subject:
// kind decides the side and the required fields.
func (a Subject) validateOperation() error {
	// Rule 3 (continued): subject.kind and subject.side must be valid too.
	if a.Kind != KindOperation &&
		a.Kind != KindRequestParameter &&
		a.Kind != KindSchema &&
		a.Kind != KindSchemaProperty {
		return fmt.Errorf("subject.kind must be a valid enum value, got %q", a.Kind)
	}
	if a.Side != SideRequest && a.Side != SideResponse {
		return fmt.Errorf("subject.side must be a valid enum value, got %q", a.Side)
	}

	// Rule 5: subject's required fields, and its side, depend on its kind.
	// Kind decides side: requests live under paths, responses under schemas.
	switch a.Kind {
	case KindOperation:
		if a.Operation == "" {
			return fmt.Errorf("subject.operation is required when kind is operation")
		}
		if a.Side != SideRequest {
			return fmt.Errorf("subject.side must be %q when kind is operation, got %q", SideRequest, a.Side)
		}
	case KindRequestParameter:
		if a.Operation == "" || a.Property == "" {
			return fmt.Errorf("subject.operation and subject.property are required when kind is request_parameter")
		}
		if a.Side != SideRequest {
			return fmt.Errorf("subject.side must be %q when kind is request_parameter, got %q", SideRequest, a.Side)
		}
	case KindSchema:
		if a.Schema == "" {
			return fmt.Errorf("subject.schema is required when kind is schema")
		}
		if a.Side != SideResponse {
			return fmt.Errorf("subject.side must be %q when kind is schema, got %q", SideResponse, a.Side)
		}
	case KindSchemaProperty:
		if a.Schema == "" || a.Property == "" {
			return fmt.Errorf("subject.schema and subject.property are required when kind is schema_property")
		}
		if a.Side != SideResponse {
			return fmt.Errorf("subject.side must be %q when kind is schema_property, got %q", SideResponse, a.Side)
		}
	}
	return nil
}
