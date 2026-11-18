// Package migrate converts stored v1 change events to the proposed v2 contract
// (the change event contract): `affected` becomes a typed
// `subject`. It runs once, at the cut-over, over every event in the store.
//
// A Stripe event keeps its fingerprint: v2's api_operation formula is v1's.
// A model retirement or an SDK release gets a new one, and `supersedes` names
// the old one, so the patch-it client, which finds its open issues and pull
// requests by the fingerprint in their marker, knows they're the same change
// and opens nothing twice.
package migrate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/patch-it-dev/patch-it-plugins/event"
	"github.com/patch-it-dev/patch-it-plugins/retire"
	"github.com/patch-it-dev/patch-it-plugins/sdkguide"
)

// providers are the model retirement slugs and who retires the models.
var providers = map[string]string{
	retire.OpenAI.API:    "openai",
	retire.Anthropic.API: "anthropic",
}

// guides are the SDKs whose v1 events came from a migration guide.
var guides = map[string]sdkguide.Guide{sdkguide.AnthropicV1.API: sdkguide.AnthropicV1}

// ToV2 converts one v1 event. It refuses an event it can't place rather than
// guess: an unknown slug with a value, or an SDK event that isn't a removed
// request parameter.
func ToV2(e event.ChangeEvent) (event.EventV2, error) {
	v2 := event.EventV2{
		ID: e.ID, API: e.API, Version: e.Version, PreviousVersion: e.PreviousVersion,
		ChangeType: e.ChangeType, Breaking: e.Breaking, Description: e.Description,
		MigrationHint: e.MigrationHint, SourceURL: e.SourceURL, SourcePointer: e.SourcePointer,
		SourceExcerpt: e.SourceExcerpt, Confidence: e.Confidence, DetectedAt: e.DetectedAt,
	}
	a := e.Affected
	if provider, ok := providers[e.API]; ok {
		if a.Value == "" {
			return event.EventV2{}, fmt.Errorf("event %d: a %s event without the model in affected.value", e.ID, e.API)
		}
		v2.Subject = event.Subject{
			Type: event.SubjectModel, Provider: provider, Name: a.Value,
			Replacement: replacement(e.MigrationHint),
		}
		return rekeyed(v2, e.Fingerprint), nil
	}
	if g, ok := guides[e.API]; ok {
		if a.Kind != event.AffectedKindRequestParameter || a.Property == "" {
			return event.EventV2{}, fmt.Errorf("event %d: an %s SDK event that isn't a removed parameter", e.ID, e.API)
		}
		v2.Subject = event.Subject{
			Type: event.SubjectPackageRelease, Ecosystem: "pypi", Package: g.API,
			Change: event.PackageRemovedParameter, Parameter: a.Property, Symbols: g.Symbols,
		}
		return rekeyed(v2, e.Fingerprint), nil
	}
	if a.Value != "" {
		return event.EventV2{}, fmt.Errorf("event %d: affected.value on %s, which isn't a model retirement", e.ID, e.API)
	}
	v2.Subject = event.Subject{
		Type: event.SubjectAPIOperation, Kind: a.Kind, Side: a.Side,
		Operation: a.Operation, Schema: a.Schema, Property: a.Property,
	}
	v2.Fingerprint = e.Fingerprint // the same formula: unchanged by construction
	return v2, nil
}

// rekeyed gives the event its v2 fingerprint and says which v1 one it replaces.
func rekeyed(v2 event.EventV2, old string) event.EventV2 {
	v2.Fingerprint = event.FingerprintV2(v2.API, v2.Version, v2.ChangeType, v2.Subject, "")
	if v2.Fingerprint != old {
		v2.Supersedes = old
	}
	return v2
}

// useInstead is the hint retire.Events writes: "Use <cell> instead: …".
var useInstead = regexp.MustCompile("^Use (.+?) instead: ")

// replacement is the one model the provider names, when its replacement cell
// is exactly one model name: not none ("names no replacement model"), not
// several, not a model with a setting (`gpt-5.6-sol` (`reasoning.mode: pro`)).
// The patch-it client opens an issue for the rest, as it does today.
func replacement(hint string) string {
	m := useInstead.FindStringSubmatch(hint)
	if m == nil {
		return ""
	}
	name := retire.First(m[1])
	if name == "" || strings.TrimSpace(m[1]) != "`"+name+"`" {
		return ""
	}
	return name
}

// Fresh converts events a source has just produced in the v1 shape. Unlike a
// stored row, a fresh event replaces nothing, so it carries no `supersedes`. An
// event that can't be placed is returned as an error, like a failed validation.
func Fresh(events []event.ChangeEvent) ([]event.EventV2, []error) {
	var out []event.EventV2
	var errs []error
	for _, e := range events {
		v2, err := ToV2(e)
		if err == nil {
			v2.Supersedes = ""
			err = v2.Validate()
		}
		if err != nil {
			errs = append(errs, err)
			continue
		}
		out = append(out, v2)
	}
	return out, errs
}
