# New Relic (`newrelic/python-agent`)

## What it watches

Functions and classes removed from New Relic's Python agent (the `newrelic` package) in 11.0.0: `newrelic.agent.record_exception`, `newrelic.agent.add_custom_parameter`, `newrelic.api.log.NewRelicLogHandler`, and others. Code that uses one breaks on the upgrade. The plugin emits one event per member, pinned to that release.

Kind: `migrationguide` (`plugins/migrationguide`), with the removed-members reader (`plugins/migrationguide/members`), reading a Markdown table.

## Why this source

- **Source:** [`python-agent-110000.mdx`](https://github.com/newrelic/docs-website/blob/f0b4f817bf9a7b76a78e26609369a8a7ce764c8e/src/content/docs/release-notes/agent-release-notes/python-release-notes/python-agent-110000.mdx) in newrelic/docs-website at commit `f0b4f817bf9a7b76a78e26609369a8a7ce764c8e` (2026-04-22, the file's latest commit). The plugin reads the raw file at that commit, because the repo has no tags. Checked on 2026-11-19. © New Relic, CC BY-NC-SA 4.0.
- **Why this one:** these are New Relic's own release notes for the agent's 11.0.0. Their "Removed features" section has a table of the APIs that 11.0.0 removed, with what to use instead. A commit never changes, so the URL is its own citation. 11.0.0 is a version a repo pins.
- **Alternatives considered:**
  - **The REST API v2 spec**, <https://api.newrelic.com/docs/swagger.yml> (OpenAPI 3.0.1, 58 operations on 43 paths, fetched 2026-11-19). New Relic serves it only live and doesn't commit it anywhere, so there's no release to pin or replay. Its deprecations aren't machine-readable: it has 0 `deprecated: true` flags. Instead, 12 operation descriptions say "This endpoint will be deprecated after", 11 of them "2024/01" and 1 "2027/01". New Relic's own end-of-life notice for REST API v2 (below) contradicts those dates for 10 of the 12: it retires them on 2027-07-31. It keeps the other 2, on `/alerts_location_failure_conditions/policies/*`, as "Not impacted".
  - **NerdGraph**, New Relic's GraphQL API. GraphQL schema changes don't fit patch-it's change-event contract, whose subjects are REST operations, models, package releases and CI runtimes.
  - **The end-of-life notices**, `src/content/eol/` in newrelic/docs-website (29 notices from 2020/05 to 2026/10, read at `219285f0d25db0d4e8127fcf9a9c3279c8bbc9b9`):
    - In 4 of the 29, `eolEffectiveDate` changed after the notice was first committed. One notice moved a month later, then two more months. One notice lost its date, which came back nine months later.
    - One notice (the file `eol-chrome-140-node-22-synthetics.md`, whose text is about the Node.js 16 and Chrome 134 synthetics runtimes) was retitled "Update Forthcoming" and now points to a later notice.
    - Only 1 of the 29 names REST operations, `eol-07-31-26-rest-api-v2.md`. The others are about agents, runtimes, images and product features, which have no operation or symbol to cite.
  - **The Node.js agent's release notes** (newrelic/node-newrelic `NEWS.md`): it removed only 1 public API method in its last 8 majors, from v6.0.0 to v14.0.0 (`setIgnoreTransaction`, in v7.0.0). That's not enough to justify a plugin.
- **Quote:** "The following APIs have been removed:" (Removed features), with rows such as "| `add_custom_parameter` | [`add_custom_attribute`](/docs/apm/agents/python-agent/python-agent-api/addcustomattribute-python-agent-api) |".

## How the provider announces changes

- Package versions (SemVer). The Python agent releases about monthly. 11.0.0 shipped on 2025-09-25 and was yanked from PyPI ("Issues with server-side configuration"). 11.0.1 (2025-10-06) has the same public API: `newrelic/agent.py` and `newrelic/api/log.py` are byte-identical at v11.0.0 and v11.0.1. The events say 11.0.0, the release that made the change.
- **Forced on a fresh install** for an unpinned or open pin (`newrelic>=10`), otherwise when the team upgrades.
- Removals are deprecated first. At v10.17.0 every removed function warns with a `DeprecationWarning`; for the two distributed-trace functions the warning is in the `Transaction` method, so it fires only inside a transaction.

## What the plugin reads and ignores

- It reads the first table under "Removed features", `| Legacy API | Replacement API |`. The table writes names bare. The plugin writes them as code reaches them after `import newrelic`: `agent.record_exception` (the functions were re-exported by `newrelic.agent`) and `api.log.NewRelicLogHandler` (never in `newrelic.agent`).
- It emits `renamed_symbol` only for a drop-in replacement, because the patch-it client applies a rename as is. A drop-in has the same arguments and behavior and already existed in `newrelic.agent` at v10.17.0 (the `DropIns` field). Every other row is a `removed_symbol`, with the "Replacement API" cell in the hint. The plugin reads a replacement written as a Markdown link around a code span (`[`name`](url)`) as that name. The 8 rows, checked against the agent's code (newrelic/newrelic-python-agent, Apache-2.0) at v10.17.0 and v11.0.0:

  | Row | Event |
  |---|---|
  | `add_custom_parameter` → `add_custom_attribute` | `renamed_symbol` `agent.add_custom_parameter` → `agent.add_custom_attribute`. At v10.17.0, `add_custom_parameter(key, value)` only warns and returns `add_custom_attribute(key, value)` |
  | `add_custom_parameters` → `add_custom_attributes` | `renamed_symbol` `agent.add_custom_parameters` → `agent.add_custom_attributes`. Same reason: `(items)` → `(items)` |
  | `accept_distributed_trace_payload` → `accept_distributed_trace_headers` | `removed_symbol`. It takes New Relic's trace payload, and the replacement takes the request's headers |
  | `create_distributed_trace_payload` → `insert_distributed_trace_headers` | `removed_symbol`. It takes nothing and returns a payload. The replacement fills a list of headers you pass it |
  | `get_browser_timing_footer` → "wrapped into `get_browser_timing_header()`" | `removed_symbol`. The cell is prose, and at v10.17.0 the footer already returns `""`. Renaming it would put the header in a page twice |
  | `record_exception` → `notice_error()` | `removed_symbol`. The arguments differ: `(exc, value, tb, params, ignore_errors)` vs `(error=(exc, value, tb), attributes=, ignore=)` |
  | `NewRelicLogHandler` → `NewRelicLogForwardingHandler` | `removed_symbol` `api.log.NewRelicLogHandler`. The old handler takes a license key, host and proxy settings and sends logs itself. The new one takes none of them |
  | `lambda_handler` & `LambdaHandlerWrapper` → moved to the `newrelic-lambda` package | Dropped and logged. The names are joined by "&", which the reader doesn't take as a list, and the replacement is another package |

  That gives 7 events.
- It ignores the section's second table, `| Legacy Setting | Replacement Setting |`. It lists `newrelic.ini` settings, not package members, and it isn't the section's first table. It also ignores the rest of the note: the "Deprecated features" settings, Python 3.7 support, and the new instrumentation.

## Known gaps

- **`lambda_handler` and `LambdaHandlerWrapper`** were both in `newrelic.agent` at v10.17.0 and are gone at v11.0.0, along with `newrelic/api/lambda_handler.py`. They get no event (see above).
- **The same functions reached another way** get no event: the `newrelic.api.transaction` and `newrelic.api.time_trace` definitions, and the methods of the same names on `Transaction` (`add_custom_parameter`, `record_exception`…). All were removed in 11.0.0 too.
- **The removed settings** (`ssl`, `error_collector.ignore_errors`, `strip_exception_messages.whitelist`…). They're configuration, not code, and change events have no subject for them.
- New Relic's REST API v2 end of life (2027-07-31) and NerdGraph (see above).
- New Relic's other agents, and later majors of the Python agent: each is a new entry in this plugin.
