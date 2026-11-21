// Package normalize turns the differ's structural changes into change events:
// the one shape every downstream step consumes, defined in
// the change event contract (contract/README.md).
//
// It is the last gate before storage. Every event carries provenance — the
// pinned spec URL, a JSON pointer, and the exact spec node the claim rests on —
// and anything that fails event.Validate is rejected rather than stored.
package normalize

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/patch-it-dev/patch-it-plugins/core/diff"
	"github.com/patch-it-dev/patch-it-plugins/core/event"
)

// excerptLimit matches the contract's cap, counted in characters.
const excerptLimit = 2000

// Source describes where a pair of specs came from, so each event can point
// back at the exact document that proves it.
type Source struct {
	API             string    // the source id: "stripe/api"
	Version         string    // info.version of the new spec
	PreviousVersion string    // info.version of the old spec
	NewSpecURL      string    // pinned URL of the new spec, e.g. .../v2468/...
	OldSpecURL      string    // pinned URL of the old spec; evidence for removals
	DetectedAt      time.Time // defaults to now
}

// Events converts changes into validated change events. A change that cannot
// be turned into a valid event is skipped and reported: the caller logs the
// reasons and exits non-zero, so a bad event is never silently dropped.
//
// Confidence is always 1.0 here. The source is a typed spec and the comparison
// is structural, so there is nothing to be unsure about: the change did happen
// exactly as described. Whether it matters to a given repository is the
// patch-it client's question, not this one's.
func Events(changes []diff.Change, src Source) ([]event.Event, []error) {
	detectedAt := src.DetectedAt
	if detectedAt.IsZero() {
		detectedAt = time.Now().UTC()
	}

	events := make([]event.Event, 0, len(changes))
	var errs []error

	for _, c := range changes {
		subject := subjectFor(c)
		sourceURL := src.NewSpecURL
		if c.InOld {
			sourceURL = src.OldSpecURL
		}

		excerpt, err := excerpt(c.Node)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s at %s: %w", c.Kind, c.Pointer, err))
			continue
		}

		e := event.Event{
			Fingerprint:     event.Fingerprint(src.API, src.Version, c.ChangeType, subject, c.Detail),
			API:             src.API,
			Version:         src.Version,
			PreviousVersion: src.PreviousVersion,
			Change:          c.ChangeType,
			Subject:         subject,
			Description:     describe(c),
			MigrationHint:   migrationHint(c),
			SourceURL:       sourceURL,
			SourcePointer:   c.Pointer,
			SourceExcerpt:   excerpt,
			Confidence:      1.0,
			DetectedAt:      detectedAt,
		}

		if err := e.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s at %s: %w", c.Kind, c.Pointer, err))
			continue
		}
		events = append(events, e)
	}

	return events, errs
}

// subjectFor maps a change's location onto the contract's four kinds. The
// side already decides the pair: requests are addressed by operation, responses
// by schema, and naming a property narrows either one.
func subjectFor(c diff.Change) event.Subject {
	a := event.Subject{
		Type:      event.SubjectAPIOperation,
		Side:      c.Side,
		Operation: c.Operation,
		Schema:    c.Schema,
		Property:  c.Property,
	}

	switch {
	case c.Side == event.SideRequest && c.Property != "":
		a.Kind = event.KindRequestParameter
	case c.Side == event.SideRequest:
		// A change to the request body's root schema has no property to name,
		// so it is reported against the operation itself.
		a.Kind = event.KindOperation
	case c.Property != "":
		a.Kind = event.KindSchemaProperty
	default:
		a.Kind = event.KindSchema
	}
	return a
}

// describe writes the one sentence a human reads first. It is generated from
// the change, never copied from the vendor's prose.
func describe(c diff.Change) string {
	where := "schema " + c.Schema
	if c.Side == event.SideRequest {
		where = c.Operation
	}

	switch c.Kind {
	case diff.OperationAdded:
		return fmt.Sprintf("New operation %s.", c.Operation)
	case diff.OperationRemoved:
		return fmt.Sprintf("Operation %s removed.", c.Operation)
	case diff.OperationDeprecated:
		return fmt.Sprintf("Operation %s marked deprecated.", c.Operation)
	case diff.SchemaAdded:
		return fmt.Sprintf("New schema %s.", c.Schema)
	case diff.SchemaRemoved:
		return fmt.Sprintf("Schema %s removed.", c.Schema)
	case diff.SchemaDeprecated:
		return fmt.Sprintf("Schema %s marked deprecated.", c.Schema)
	case diff.PropertyAdded:
		return fmt.Sprintf("Property %s added to %s.", c.Property, where)
	case diff.PropertyRemoved:
		return fmt.Sprintf("Property %s removed from %s.", c.Property, where)
	case diff.PropertyDeprecated:
		return fmt.Sprintf("Property %s of %s marked deprecated.", c.Property, where)
	case diff.BecameRequired:
		if c.Detail == "added as required" {
			return fmt.Sprintf("New required property %s in %s.", c.Property, where)
		}
		return fmt.Sprintf("Property %s is now required in %s.", c.Property, where)
	case diff.BecameOptional:
		return fmt.Sprintf("Property %s is no longer required in %s.", c.Property, where)
	case diff.EnumValueAdded:
		return fmt.Sprintf("Enum value %q added to %s in %s.", c.Detail, c.Property, where)
	case diff.EnumValueRemoved:
		return fmt.Sprintf("Enum value %q removed from %s in %s.", c.Detail, c.Property, where)
	case diff.MemberAdded:
		return fmt.Sprintf("Variant %s added to %s in %s.", c.Detail, c.Property, where)
	case diff.MemberRemoved:
		return fmt.Sprintf("Variant %s removed from %s in %s.", c.Detail, c.Property, where)
	case diff.TypeChanged:
		return fmt.Sprintf("Type of %s in %s changed: %s.", c.Property, where, c.Detail)
	default:
		return fmt.Sprintf("%s at %s.", c.Kind, c.Pointer)
	}
}

// migrationHint is filled only when the change itself states the fix. A removed
// property does not say what replaced it, and guessing at renames would be
// exactly the invention this pipeline exists to avoid, so those stay empty.
func migrationHint(c diff.Change) string {
	switch c.Kind {
	case diff.BecameRequired:
		if c.Side == event.SideRequest {
			return fmt.Sprintf("Always send %s.", c.Property)
		}
	case diff.EnumValueRemoved:
		if c.Side == event.SideRequest {
			return fmt.Sprintf("Stop sending %q for %s.", c.Detail, c.Property)
		}
	case diff.MemberRemoved:
		if c.Side == event.SideRequest {
			return fmt.Sprintf("Stop sending the %s variant of %s.", c.Detail, c.Property)
		}
	case diff.TypeChanged:
		return fmt.Sprintf("Handle %s as %s.", c.Property, strings.TrimSpace(afterArrow(c.Detail)))
	}
	return ""
}

// afterArrow pulls the new value out of a "before → after" detail.
func afterArrow(detail string) string {
	if _, after, found := strings.Cut(detail, "→"); found {
		return after
	}
	return detail
}

// excerpt serializes the spec node the pointer addresses. encoding/json sorts
// map keys, so the same node always produces the same text. Long nodes are cut
// to the contract's limit, counted in characters.
func excerpt(node any) (string, error) {
	if node == nil {
		return "", fmt.Errorf("no spec node to quote as evidence")
	}
	data, err := json.Marshal(node)
	if err != nil {
		return "", fmt.Errorf("serializing evidence: %w", err)
	}
	text := string(data)
	if utf8.RuneCountInString(text) <= excerptLimit {
		return text, nil
	}
	runes := []rune(text)
	return string(runes[:excerptLimit-1]) + "…", nil
}
