package event

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type (
	ChangeType   string
	AffectedKind string
	Side         string
)

const (
	ChangeTypeBreaking    ChangeType = "breaking"
	ChangeTypeDeprecation ChangeType = "deprecation"
	ChangeTypeNewFeature  ChangeType = "new_feature"
	ChangeTypeFix         ChangeType = "fix"
)

const (
	AffectedKindOperation        AffectedKind = "operation"
	AffectedKindRequestParameter AffectedKind = "request_parameter"
	AffectedKindSchema           AffectedKind = "schema"
	AffectedKindSchemaProperty   AffectedKind = "schema_property"
)

const (
	SideRequest  Side = "request"
	SideResponse Side = "response"
)

type Affected struct {
	Kind      AffectedKind `json:"kind"`                // kind of change
	Side      Side         `json:"side"`                // which side is breaking
	Operation string       `json:"operation,omitempty"` // if operation change, what changed
	Schema    string       `json:"schema,omitempty"`    // if schema change, what changed
	Property  string       `json:"property,omitempty"`  // if poperty change, what changed
	// Value is the one value of Property the change is about, when it's about one:
	// the retired model name for a model retirement ("gpt-4"). Optional.
	Value string `json:"value,omitempty"`
}

type ChangeEvent struct {
	ID              int64      `json:"id"`
	Fingerprint     string     `json:"fingerprint"`
	API             string     `json:"api"`
	Version         string     `json:"version"`
	PreviousVersion string     `json:"previous_version,omitempty"`
	ChangeType      ChangeType `json:"change_type"`
	Breaking        bool       `json:"breaking"`
	Affected        Affected   `json:"affected"`
	Description     string     `json:"description"`
	MigrationHint   string     `json:"migration_hint,omitempty"`
	SourceURL       string     `json:"source_url"`
	SourcePointer   string     `json:"source_pointer,omitempty"`
	SourceExcerpt   string     `json:"source_excerpt"`
	Confidence      float64    `json:"confidence"`
	DetectedAt      time.Time  `json:"detected_at"`
}

// Fingerprint identifies one change at one location, so re-running ingest over
// the same specs updates nothing instead of duplicating events. detail is what
// separates two changes at the same place: the enum value added, the old and
// new type, "added as required". Absent fields are empty strings, and the parts
// are joined with a separator that cannot appear in a path or a schema name.
func Fingerprint(api, version string, changeType ChangeType, affected Affected, detail string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		api,
		version,
		string(changeType),
		string(affected.Kind),
		string(affected.Side),
		affected.Operation,
		affected.Schema,
		affected.Property,
		detail,
	}, "|")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// fingerprintPattern is the contract's shape for a fingerprint (the schema's
// pattern in the change event contract). patch-it checks the same pattern.
var fingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func (e ChangeEvent) Validate() error {
	if err := checkCommon(e.SourceURL, e.SourceExcerpt, e.Fingerprint, e.API, e.Version,
		e.Description, e.DetectedAt, e.ChangeType, e.Breaking, e.Confidence); err != nil {
		return err
	}

	if err := e.Affected.validate(); err != nil {
		return err
	}

	// Rule 5 (continued): a value is a value of a property.
	if e.Affected.Value != "" && e.Affected.Property == "" {
		return fmt.Errorf("affected.value needs affected.property: it is a value of that property")
	}

	return nil
}

// checkCommon is the contract's rules for every field outside affected (v1) or
// subject (v2): the two versions differ only in what changed, not in the rest.
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

// validate is the contract's rules for an API location: v1's affected, and v2's
// api_operation subject. Kind decides the side and the required fields.
func (a Affected) validate() error {
	// Rule 3 (continued): affected.kind and affected.side must be valid too.
	if a.Kind != AffectedKindOperation &&
		a.Kind != AffectedKindRequestParameter &&
		a.Kind != AffectedKindSchema &&
		a.Kind != AffectedKindSchemaProperty {
		return fmt.Errorf("affected.kind must be a valid enum value, got %q", a.Kind)
	}
	if a.Side != SideRequest && a.Side != SideResponse {
		return fmt.Errorf("affected.side must be a valid enum value, got %q", a.Side)
	}

	// Rule 5: affected's required sub-fields, and its side, depend on its kind.
	// Kind decides side: requests live under paths, responses under schemas.
	switch a.Kind {
	case AffectedKindOperation:
		if a.Operation == "" {
			return fmt.Errorf("affected.operation is required when kind is operation")
		}
		if a.Side != SideRequest {
			return fmt.Errorf("affected.side must be %q when kind is operation, got %q", SideRequest, a.Side)
		}
	case AffectedKindRequestParameter:
		if a.Operation == "" || a.Property == "" {
			return fmt.Errorf("affected.operation and affected.property are required when kind is request_parameter")
		}
		if a.Side != SideRequest {
			return fmt.Errorf("affected.side must be %q when kind is request_parameter, got %q", SideRequest, a.Side)
		}
	case AffectedKindSchema:
		if a.Schema == "" {
			return fmt.Errorf("affected.schema is required when kind is schema")
		}
		if a.Side != SideResponse {
			return fmt.Errorf("affected.side must be %q when kind is schema, got %q", SideResponse, a.Side)
		}
	case AffectedKindSchemaProperty:
		if a.Schema == "" || a.Property == "" {
			return fmt.Errorf("affected.schema and affected.property are required when kind is schema_property")
		}
		if a.Side != SideResponse {
			return fmt.Errorf("affected.side must be %q when kind is schema_property, got %q", SideResponse, a.Side)
		}
	}
	return nil
}
