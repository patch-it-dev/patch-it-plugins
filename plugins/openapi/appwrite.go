package openapi

import "github.com/patch-it-dev/patch-it-plugins/diff"

// appwriteDiff is how Appwrite's spec says what it says:
//
//   - A deprecation is the `deprecated` flag plus x-appwrite.deprecated,
//     {"since": "1.8.0", "replaceWith": "tablesDB.createTransaction"}: the
//     only place the spec says what to use instead (the description doesn't).
//     x-appwrite sorts last in the operation, after parameters and
//     responses, so quoting the whole operation can lose it to the
//     2,000-character cut: the event quotes x-appwrite.deprecated itself.
//     The flag without it is quoted as the flag (never seen so far).
//   - The flag also marks an operation one of whose SDK method names was
//     renamed: POST /account/mfa/challenges is flagged with "replaceWith":
//     "account.createMFAChallenge", which is the same operation's other
//     name in x-appwrite.methods (createMfaChallenge became
//     createMFAChallenge). The route isn't deprecated, so that flag isn't
//     read as a deprecation: see appwriteMethodRename.
//   - Path parameters are matched by place, as for the other specs.
var appwriteDiff = diff.Options{
	PathTemplates:      true,
	QuoteDeprecated:    "/x-appwrite/deprecated",
	FlagNotDeprecation: appwriteMethodRename,
}

// appwriteMethodRename reports whether an operation's deprecation is the
// rename of one of its SDK methods: x-appwrite.deprecated.replaceWith names
// a method of the same operation's x-appwrite.methods
// ("<namespace>.<name>"). 23 of the 99 operations flagged in 2.7.x's server
// operations are, all flagged since 1.8.0 (1.8.x's server document at
// ac705cd3 has the same 23).
func appwriteMethodRename(op map[string]any) bool {
	xa, _ := op["x-appwrite"].(map[string]any)
	dep, _ := xa["deprecated"].(map[string]any)
	replaceWith, _ := dep["replaceWith"].(string)
	if replaceWith == "" {
		return false
	}
	methods, _ := xa["methods"].([]any)
	for _, m := range methods {
		method, _ := m.(map[string]any)
		namespace, _ := method["namespace"].(string)
		name, _ := method["name"].(string)
		if name != "" && namespace+"."+name == replaceWith {
			return true
		}
	}
	return false
}

// appwriteKeep keeps operation deprecations only: not even a deprecated
// operation's removal. Read as Fetch reads it (the newest minor's server
// operations at each of the 139 commits on appwrite/specs' main, 2026-02-17
// to 2026-11-17), the spec's history has 14 deprecations, all on
// 2026-06-24 (see the golden test), and 78 operations leaving it (from the spec, or from the server
// platform), only one
// marked deprecated first, and that one a path move: PATCH
// /databases/{databaseId}/collections/{collectionId}/attributes/{key}/relationship
// became .../attributes/relationship/{key} on 2026-03-20. Of the others, 13
// are back in the spec now: 2.4.x's single document took the server platform
// off them on 2026-10-12 as not yet in a released SDK, and 2.5.x had them
// on it again the next day. Nothing in the spec says when an operation stops answering, so a
// removal isn't an event.
//
// Property changes and new operations are dropped too, as for vercel.
func appwriteKeep(c diff.Change) bool { return c.Kind == diff.OperationDeprecated }
