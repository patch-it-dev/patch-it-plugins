package openapi

import "github.com/patch-it-dev/patch-it-plugins/core/diff"

// elasticsearchDiff reads Elastic's spec like Supabase's: deprecations by the
// `deprecated` flag, quoting the whole operation, and path templates matched.
var elasticsearchDiff = diff.Options{PathTemplates: true, QuoteDeprecatedOperation: true}

// elasticsearchKeep keeps operation deprecations, and removals of operations
// marked deprecated first. Over the minor versions with an OpenAPI output
// (8.14 to 9.6, compared in version order) that's 23 deprecations and 3
// removals, all in 9.0: GET and POST /{index}/_knn_search, and POST
// /{index}/_unfreeze ("Remove unfreeze REST endpoint", Elastic's 9.0.0
// breaking changes). The 31 other removals weren't deprecated first and
// mostly weren't removals from Elasticsearch:
//
//   - APIs a later minor of the older major had and the newer major's
//     branch, cut earlier, didn't yet: 8.19 to 9.0 drops
//     /_data_stream/{name}/_options, which is back in 9.1;
//   - APIs Elastic documents as not for direct use ("designed for indirect
//     use by Elasticsearch Service, Elastic Cloud Enterprise, and Elastic
//     Cloud on Kubernetes. Direct use is not supported."), left out of the
//     spec from 9.2: /_autoscaling, /_nodes/{node_id}/shutdown.
var elasticsearchKeep = deprecationsOnly(elasticsearchDiff)
