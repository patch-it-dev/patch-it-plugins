package openapi

import "github.com/patch-it-dev/patch-it-plugins/diff"

// algoliaDiff reads Algolia's specs like Supabase's: deprecations by the
// `deprecated` flag, quoting the whole operation, and path templates matched.
var algoliaDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// algoliaKeep keeps operation deprecations, and removals of operations
// marked deprecated first, of REST operations only. Algolia's specs also
// describe what only its API clients do, and a change to one of those isn't
// an API change:
//
//   - helpers, marked `x-helper: true`: /saveObjects, /waitForTask,
//     /replaceAllObjects, … (16 in the Search API on 2026-11-17);
//   - the "custom request" methods for any path, operationId customGet,
//     customPost, customPut and customDelete on /{path} (before 2024-03-25,
//     /1{path}).
//
// In every release from 2023-08-22 to 2026-11-17 this keeps 9 Search API
// deprecations (the multi-cluster /1/clusters endpoints, 2025-04-01) and 9
// Ingestion API ones (the /1/tasks endpoints, 2025-01-22, and PUT
// /1/tasks/{taskID}/disable, 2024-07-26). None of the 17 removals in that
// history was deprecated first; 13 were the client-only entries above (the
// custom-request methods in all three specs on 2024-03-25, and /saveObjects).
func algoliaKeep(c diff.Change) bool {
	// c.Node is the operation only because algoliaDiff sets
	// QuoteDeprecatedOperation (a removal carries the old operation anyway);
	// TestAlgoliaClientOnlyFilterThroughTheDiffer pins that.
	op, _ := c.Node.(map[string]any)
	if clientOnly(op) {
		return false
	}
	return deprecationsOnly(algoliaDiff)(c)
}

// clientOnly is an Algolia spec entry for an API-client method, not a REST
// operation.
func clientOnly(op map[string]any) bool {
	if helper, _ := op["x-helper"].(bool); helper {
		return true
	}
	id, _ := op["operationId"].(string)
	switch id {
	case "customGet", "customPost", "customPut", "customDelete":
		return true
	}
	return false
}
