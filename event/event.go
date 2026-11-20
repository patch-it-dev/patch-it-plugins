// Package event is the old import path of github.com/patch-it-dev/patch-it-plugins/core/event.
//
// Deprecated: import github.com/patch-it-dev/patch-it-plugins/core/event
// instead. This package only forwards to it, so code written against the old
// path keeps building; it will be removed in a later release.
package event

import moved "github.com/patch-it-dev/patch-it-plugins/core/event"

// Deprecated: the types live in github.com/patch-it-dev/patch-it-plugins/core/event.
type (
	Affected      = moved.Affected
	AffectedKind  = moved.AffectedKind
	ChangeEvent   = moved.ChangeEvent
	ChangeType    = moved.ChangeType
	EventV2       = moved.EventV2
	PackageChange = moved.PackageChange
	Side          = moved.Side
	Subject       = moved.Subject
	SubjectType   = moved.SubjectType
)

// Deprecated: the constants live in github.com/patch-it-dev/patch-it-plugins/core/event.
const (
	AffectedKindOperation        = moved.AffectedKindOperation
	AffectedKindRequestParameter = moved.AffectedKindRequestParameter
	AffectedKindSchema           = moved.AffectedKindSchema
	AffectedKindSchemaProperty   = moved.AffectedKindSchemaProperty
	ChangeTypeBreaking           = moved.ChangeTypeBreaking
	ChangeTypeDeprecation        = moved.ChangeTypeDeprecation
	ChangeTypeFix                = moved.ChangeTypeFix
	ChangeTypeNewFeature         = moved.ChangeTypeNewFeature
	PackageChangedDefault        = moved.PackageChangedDefault
	PackageRemovedParameter      = moved.PackageRemovedParameter
	PackageRemovedSymbol         = moved.PackageRemovedSymbol
	PackageRenamedSymbol         = moved.PackageRenamedSymbol
	SideRequest                  = moved.SideRequest
	SideResponse                 = moved.SideResponse
	SubjectAPIOperation          = moved.SubjectAPIOperation
	SubjectCIRuntime             = moved.SubjectCIRuntime
	SubjectModel                 = moved.SubjectModel
	SubjectPackageRelease        = moved.SubjectPackageRelease
)

// Deprecated: the functions and values live in github.com/patch-it-dev/patch-it-plugins/core/event.
var (
	DecodeV2      = moved.DecodeV2
	Fingerprint   = moved.Fingerprint
	FingerprintV2 = moved.FingerprintV2
)
