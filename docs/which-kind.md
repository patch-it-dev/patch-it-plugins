# Which kind fits my provider?

A provider has one or more sources: each place it publishes changes. Each source is read by one **kind**. Answer the questions below for one source, top to bottom; the first "yes" is its kind. A provider with several sources (an API spec and an SDK, say) asks them once per source.

The list of what's already covered isn't here: `make plugins` prints it from the code.

Every kind is one of two shapes ([writing-a-plugin.md, step 4](writing-a-plugin.md#4-the-interface)): **versioned** (it lists versions and reads each one: `openapi`, `migrationguide`, `changelog`, `catalog`, `apisurface`) or **living** (a page read as it is now, each read that changed a new dated version: `notices`).

| Ask | If yes, the kind | It reads | Examples of the shape |
|---|---|---|---|
| Does the provider publish an **OpenAPI spec** of its API, per release or per commit? | `openapi` | Two specs, diffed: removed and renamed operations, parameters, fields; deprecations | A spec repo with release tags; a spec file a repo commits on every change |
| Is there a **guide that lists what a release removed**, one per major or per release, in a table? | `migrationguide` | The guide at the release's tag: removed request parameters (reader `sdk`) or removed members of a module (reader `members`, RST or Markdown tables) | `MIGRATION.md` per major; "removed members" tables in a 2.0 migration guide; an API version removal guide |
| Is there a **changelog**: a file per release (`CHANGELOG.md`, `CHANGES.rst`, `HISTORY.rst`), release notes on each GitHub release, or a machine-readable JSON changelog? | `changelog` | The release's section at its tag, or the tag's GitHub release notes (`Notes: true`) (reader `markdown`), or the life-cycle entries a JSON changelog adds commit by commit (reader `json`) | Keep a Changelog files; release-please and changesets changelogs; reStructuredText release notes; an SDK with only GitHub release notes; an API changelog JSON in a repo; a Terraform provider's changelog |
| Is there a **live page of dated notices**: a deprecations or shutdown table, an end-of-life table, or an **RSS/Atom feed** of announcements? | `notices` | The page as it is now, cited through a Web Archive snapshot, against the snapshot kept from the read before: a living page, with no versions of its own (reader `deprecations`, `feed`, or an end-of-life table) | A model deprecations page; an API end-of-life policy table; a changelog feed with a "deprecation" label; a Kubernetes API removals page |
| Is there a **structured catalog** in the provider's repo: one JSON file per command, or one file of upcoming removals? | `catalog` | Each release (or commit) of the catalog, compared with the one before: names that disappeared entirely or are new (added), or removals newly announced | The command definitions a server is built from, at each release tag |
| Is it a **published package** with no changelog worth reading? | `apisurface` | Each release's files from the registry (PyPI wheels, npm TypeScript declarations), compared: removed or renamed classes, functions, methods, and parameters removed or newly required; new ones, and new optional parameters, as additions | An SDK generated from a spec, released often, whose changelog says "chore: update types" |

Still no "yes"? Open a [discussion](https://github.com/patch-it-dev/patch-it-plugins/discussions) with the source's link: a new kind is a new package under `kinds/` (how to read, written once; each provider's sources in `providers/<provider>/` say what to read), and the questions above get one more row.

## Two sources, not one kind

- **An API and its SDKs are separate sources.** The API's changes come from its spec or its notices; an SDK's from its migration guide, its changelog, or its published files. Register each, on its kind, in the provider's one `providers/<provider>/provider.go`.
- **There is no kind per vendor or per tool.** Kubernetes API removals are a `migrationguide` or `notices` source; a Kubernetes client is a package source (`changelog` or `apisurface`); a Terraform provider is a `changelog` source.
- **Prefer the most machine-readable source** (a spec beats a changelog, a changelog beats a feed, a feed beats a blog post): [writing-a-plugin.md, step 2](writing-a-plugin.md#2-pick-the-source).

Then scaffold it: `make new-plugin ARGS='--provider <id> --source <id> --kind <kind>'` ([step 1](writing-a-plugin.md#1-scaffold-it-with-new-plugin)).
