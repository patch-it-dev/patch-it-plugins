package event

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Change event v2 (the change event contract), in force since the
// cut-over: v1's fields with `affected`, which can only describe an OpenAPI
// location, replaced by a typed `subject`. The store holds and serves v2; the
// sources still build v1 events, converted on the way out (migrate/).

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

	// api_operation: v1's affected, without value (a model has its own type).
	Kind      AffectedKind `json:"kind,omitempty"`
	Side      Side         `json:"side,omitempty"`
	Operation string       `json:"operation,omitempty"`
	Schema    string       `json:"schema,omitempty"`
	Property  string       `json:"property,omitempty"`

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

// EventV2 is a v2 change event.
type EventV2 struct {
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

// DecodeV2 reads one v2 event and refuses a field the contract doesn't define,
// at the top level or in subject: the patch-it client's model forbids extra
// fields too, so the hosted service and the client reject the same events. Keys are
// matched exactly ("TYPE" isn't "type", though encoding/json would take it),
// and a subject key belongs to its type even when its value is empty: pydantic
// sees the key, not the value.
func DecodeV2(raw []byte) (EventV2, error) {
	var e EventV2
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		return EventV2{}, err
	}
	var keys struct {
		Top     map[string]json.RawMessage
		Subject map[string]json.RawMessage `json:"subject"`
	}
	if err := json.Unmarshal(raw, &keys.Top); err != nil {
		return EventV2{}, err
	}
	for k := range keys.Top {
		if !slices.Contains(eventKeys, k) {
			return EventV2{}, fmt.Errorf("%q isn't a field of a change event", k)
		}
	}
	if sub, ok := keys.Top["subject"]; ok {
		if err := json.Unmarshal(sub, &keys.Subject); err != nil {
			return EventV2{}, err
		}
	}
	allowed := slices.Concat(subjectKeys[e.Subject.Type], []string{"type"})
	for k := range keys.Subject {
		if !slices.Contains(allowed, k) {
			return EventV2{}, fmt.Errorf("subject.%s isn't a field of type %s", k, e.Subject.Type)
		}
	}
	return e, nil
}

// eventKeys are the JSON names of EventV2's fields.
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

// FingerprintV2 identifies one change. For api_operation it is v1's formula
// byte for byte, so every stored Stripe event keeps its fingerprint. For the
// other types, the type's key fields stand where v1's location was and detail
// is empty: the key already says what changed. Symbols are sorted first, so
// the same release always gets the same fingerprint.
func FingerprintV2(api, version string, changeType ChangeType, s Subject, detail string) string {
	if s.Type == SubjectAPIOperation {
		return Fingerprint(api, version, changeType, Affected{
			Kind: s.Kind, Side: s.Side, Operation: s.Operation, Schema: s.Schema, Property: s.Property,
		}, detail)
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
	sum := sha256.Sum256([]byte(strings.Join(append(parts, ""), "|")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Validate applies the v2 contract: v1's rules for everything outside subject,
// then subject's own.
func (e EventV2) Validate() error {
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
		// v1's kind and side rules, unchanged.
		a := Affected{Kind: s.Kind, Side: s.Side, Operation: s.Operation, Schema: s.Schema, Property: s.Property}
		if err := a.validate(); err != nil {
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
