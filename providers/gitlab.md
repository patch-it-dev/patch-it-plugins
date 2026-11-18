# GitLab REST API (`gitlab`)

## What it watches

GitLab's REST API (`/api/v4`) between releases, **request side only**: endpoints removed, and parameters callers send that are removed, renamed or newly required.

Format plugin: `openapi` (`plugins/openapi`), with `adapter.GitLab` and a request-side filter (`openapi.Plugin.Keep`).

## Why this source

- **Source:** `doc/api/openapi/openapi_v3.yaml` in [gitlab-org/gitlab](https://gitlab.com/gitlab-org/gitlab) at a release tag, read through GitLab's Repository Files API (`https://gitlab.com/api/v4/projects/gitlab-org%2Fgitlab/repository/files/doc%2Fapi%2Fopenapi%2Fopenapi_v3.yaml/raw?ref=<tag>`); the newest stable `vX.Y.Z-ee` tag from the tags API. Checked on 2026-11-15.
- **Why this one:** GitLab generates it from its own API code at every release, and a tag never changes, so two releases diff exactly. The Files API is GitLab's documented interface for programs and returns the same bytes as the web's raw route.
- **Alternatives considered:** the [deprecations page](https://docs.gitlab.com/update/deprecations/) and its YAML sources (dated and structured, but *what* changes is prose); the [breaking-changes Atom feed](https://docs.gitlab.com/releases/breaking-changes.xml) (same prose); the REST deprecations page (changes deferred to a v5 that isn't being built); `openapi_v2.yaml` (Swagger 2.0, partial, superseded).
- **Quote:** "From GitLab 19.0, if you want to import repositories from Bitbucket Cloud through the GitLab API, you must use user API tokens instead." ([deprecations page](https://docs.gitlab.com/update/deprecations/)).

## How the provider announces changes

- REST is versioned by path (`/api/v4`), never by date or header. Removals ship in major releases (18.0, 19.0) and are listed on the deprecations page beforehand.
- **Forced**: on GitLab.com when the release ships; on self-managed instances when they upgrade, so the instance's version is the pin.
- A minor release every month, patch releases in between. The event's `version` is the GitLab version (`19.0.0`), not the tag (`v19.0.0-ee`) and not `info.version` (`v4`).

## What the plugin reads and ignores

- **Response-side changes are dropped.** Between 18.11 and 19.0 the generator retyped hundreds of response properties (default `string` to the real type) while GitLab returned the same JSON; reporting them would be wrong pull requests. Precision over recall: only request-side changes are kept.
- Request bodies are named by a hash of their content (`RequestBody_<hex>`), so adding one field renames the schema. The adapter inlines those bodies back under their operations, and each event's pointer still names the node in the published file.
- Release candidates are skipped; tags are sorted numerically.
- The OpenAPI 3 file exists from `v18.10.0-ee`; 19.0 is the first major release that can be replayed (`Replay` from `v18.11.0-ee` to `v19.0.0-ee`).

## Known gaps

- Real response-side removals are missed (the deprecations page would recover them; not built).
- GraphQL removals aren't watched.
