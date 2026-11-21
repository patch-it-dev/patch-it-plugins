# OpenAI (`openai/api`)

## What it watches

The OpenAI REST API (`api.openai.com/v1`): operations OpenAI marks deprecated, and removals of operations it had marked deprecated, day by day in the OpenAPI spec its SDKs are generated from.

Kind: `openapi` (`plugins/openapi`), with `adapter.OpenAI`.

## Why this source

- **Source:** `openapi.yaml` in [openai/openai-openapi](https://github.com/openai/openai-openapi) (MIT), pinned to a commit (`https://raw.githubusercontent.com/openai/openai-openapi/<sha>/openapi.yaml`). The versions come from the GitHub commits API. Checked on 2026-11-21.
- **Why this one:**
  - It's OpenAI's own description of the API, the one `openai-python` and `openai-node` are generated from, and it's kept current: 589 commits changed it from 2022-01-19 to 2026-11-21, several on most days.
  - It marks deprecated operations with the `deprecated` flag, and the operation's description says until when.
  - Every commit can be cited and replayed.
- **Alternatives considered:**
  - **The repo's tags** (`1.3.0`, `2.0.0`): too sparse to follow the API.
  - **`openapi.json` beside it:** the same spec, but only committed from 2026-07-16 on; the YAML has the whole history.
  - **The [deprecations page](https://developers.openai.com/api/docs/deprecations):** read for model shutdowns by `openai/models` (kind `notices`). It announces endpoint shutdowns in prose and tables, not per operation.
  - **The changelog:** product news, not keyed by operation.
- **Quote:** "The endpoint and model are deprecated and scheduled to shut down on February 26, 2027." (`POST /audio/translations`, flagged deprecated on 2026-10-28)

## How the provider announces changes

- **One version per day.** The spec is committed several times a day, and `info.version` (`2.3.0`) isn't bumped with each change, so a version is a day: the day's last commit, its date the event's `version`.
- **Operations are named by the path a caller sends** (`POST /v1/audio/translations`): the server is `https://api.openai.com/v1` and the paths leave out `/v1`, so the adapter moves it into the paths.
- **Deprecations use the flag.** On 2026-11-21, 17 operations carry it, and none is called deprecated only in words (one description mentions deprecated models, not the operation). Flagged descriptions give the shutdown date.
- **Much of the daily churn is the spec's own shape**: enum lists rewritten as `const`, schemas regenerated, work in progress added and pulled again.

## What the plugin reads and ignores

- **Reads** every day's spec, diffed against the day before. Two events come out of it:
  - **Deprecations:** an operation that gains the `deprecated` flag. The event quotes the whole operation, so the shutdown date is in the excerpt.
  - **Removals (breaking):** an operation that leaves the spec after the previous day's spec had marked it deprecated.
- **Matches operations by path template,** so a renamed path parameter isn't a removal.
- **Ignores everything else:** removals of operations that were never deprecated, schema and property changes, new operations. The drop test replays 2026-11-17 → 2026-11-18, when the session spending-limit schemas added the day before were pulled again: three schema removals, no event.
- **The golden test** replays 2026-10-27 → 2026-10-28: `POST /audio/translations` deprecated, one event; the two new `/organization/external_keys` operations of the same day are dropped.

## Known gaps

- **Deprecated parameters, fields and enum values** (`user` in favor of `safety_identifier`, a model in a `model` enum) aren't events: only whole operations are.
- **Removals OpenAI didn't flag first aren't events,** even if the endpoint really went away.
- **History:** the GitHub commits API lists at most 1,000 commits. The spec has 589 (back to 2022-01-19), so all are listed today; at its pace, the oldest will start dropping off in 2027.
- **The patch-it client needs `openai` in its list of date-versioned APIs** to place the events' date `version`; that list is outside this repo.
