# Elasticsearch (`elastic/elasticsearch-api`)

## What it watches

The Elasticsearch REST API, minor version by minor version: operations Elastic marks deprecated, and removals of operations it had marked deprecated, between two versions of the OpenAPI spec generated from Elastic's API specification.

Kind: `openapi` (`plugins/openapi`), with `adapter.Elasticsearch`.

## Why this source

- **Source:** `output/openapi/elasticsearch-openapi.json` in [elastic/elasticsearch-specification](https://github.com/elastic/elasticsearch-specification), on the branch of the newest minor version Elastic has released, pinned to that branch's head commit (`https://raw.githubusercontent.com/elastic/elasticsearch-specification/<sha>/output/openapi/elasticsearch-openapi.json`).
  - The newest released version comes from [elastic/elasticsearch](https://github.com/elastic/elasticsearch)'s releases. The branch list comes from the GitHub refs API.
  - Checked on 2026-11-19.
- **Why this one:** the specification is what Elastic generates its official clients and its API reference from.
  - It has one branch per minor version, and the OpenAPI output exists from 8.14 on.
  - Deprecations use the `deprecated` flag.
  - Its removals of deprecated operations match Elastic's release notes: the 9.0 spec drops `POST /{index}/_unfreeze`, and the [9.0.0 breaking changes](https://www.elastic.co/docs/release-notes/elasticsearch/breaking-changes) say "Remove unfreeze REST endpoint".
- **Alternatives considered:**
  - **The [breaking changes](https://www.elastic.co/docs/release-notes/elasticsearch/breaking-changes) and [deprecations](https://www.elastic.co/docs/release-notes/elasticsearch/deprecations) pages,** served as Markdown with one section per release. They're prose grouped by area, and most bullets are about settings, analyzers, privileges or behavior ("Set allow_partial_search_results=true by default"), not an endpoint. That makes them a check on the spec rather than a source to read events from.
  - **`output/schema/schema.json` in the same repo.** It's the specification's own model, of which the OpenAPI output is a rendering.
  - **The clients' changelogs** (`elasticsearch-py`, `elasticsearch-js`). They're generated from the same specification.
- **Quote:** "Remove unfreeze REST endpoint" (Elastic's 9.0.0 breaking changes; `POST /{index}/_unfreeze`, deprecated in the 8.19 spec, is gone from 9.0's)

## How the provider announces changes

- **Versions are the product's:** minor releases (9.5), patch releases on each, and two majors maintained side by side (8.19 patches came out after 9.5's). The event's `version` is the minor's first release (`9.0.0`). That's also what a repo pins: the clients' versions follow the server's (`elasticsearch-py` 9.5.1 for Elasticsearch 9.5).
- **An API is deprecated in one version and removed in a later major,** typically.

## What the plugin reads and ignores

- **Reads:** the spec at the head of the newest released minor's branch, diffed against the last one read. That covers fixes landing on the branch and, once Elastic releases the next minor, the move to its branch. The next, unreleased minor (9.6 on 2026-11-19) isn't read: its spec can still change before it ships.
- **Two events come out of it:**
  - **Deprecations:** an operation that gains the `deprecated` flag, quoting the whole operation.
  - **Removals (breaking):** an operation that leaves the spec after the previous version had marked it deprecated.
- **Matches operations by path template.**
- **Counts a removal only from one minor to the next.** A release branch's spec gets fixes after its .0 release (the 9.2 branch dropped `GET /_project/tags` after 9.2.0), and Elasticsearch removes APIs in a new minor or major, never in a patch. So a removal between two commits of one branch is a correction to the spec, even of a deprecated operation, and isn't an event. A deprecation on a branch after its .0 is still one, dated by that minor's `.0`.
- **Ignores everything else:**
  - **Removals of operations that were never deprecated.** 31 of them over 8.14 to 9.6 (compared in version order), and they mostly weren't removals from Elasticsearch:
    - APIs a later minor of the older major has and the newer major's branch, cut earlier, didn't yet. 8.19 to 9.0 drops `/_data_stream/{name}/_options`, which is back in 9.1.
    - APIs Elastic documents as not for direct use ("designed for indirect use by Elasticsearch Service, Elastic Cloud Enterprise, and Elastic Cloud on Kubernetes. Direct use is not supported."), left out of the spec from 9.2: `/_autoscaling`, `/_nodes/{node_id}/shutdown`.
  - **Property changes, and new operations.**
- **History:** over 8.14 to 9.6, this keeps 26 events.
  - **23 deprecations.** For example, the rollup and legacy `_template` endpoints (8.17), `/_application/analytics` (9.0), and `POST /_ml/anomaly_detectors/{job_id}/_flush` (9.1).
  - **3 removals, in 9.0:** GET and POST `/{index}/_knn_search`, and POST `/{index}/_unfreeze`.
- **Reads** 8.19 → 9.0 in the golden test. The drop test replays 9.1 → 9.2.

## Known gaps

- **Only from 8.14,** when the specification started generating OpenAPI.
- **Deprecations announced only in the release notes aren't events.** That includes settings, parameters described in prose, and behavior changes.
- **A "removed" event means removed from the spec.** An operation dropped from the spec while Elasticsearch still serves it would read as removed if it had been marked deprecated. `vercel`, `supabase` and the `algolia-*` plugins share this.
- **The two maintained majors are read as one line.** When the newest release is a 9.x minor, an 8.19-only deprecation isn't read.
- **Request and response property changes aren't events,** including deprecated parameters.
