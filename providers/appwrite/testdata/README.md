# Appwrite: saved copies and goldens

Real copies of what Appwrite publishes, read by this folder's tests instead of the network: each row says where it was fetched, when, its license and a quote. The golden files under `golden/` are every event the tests must store; `go test ./providers/appwrite -update` rewrites them after an intended change, and the PR says why the new output is right.

## Saved OpenAPI spec excerpts

Each pair is a real breaking change between two releases of the provider's own spec, trimmed to the schemas and paths involved: every line is copied unchanged, the rest of the spec was left out. The tests replay one release to the next and check the events.

| Files | Source | Checked | License | Quote |
|---|---|---|---|---|
| `appwrite-1.9.x-8c00ba35-server.json`, `appwrite-1.9.x-7d74bf9f-server.json` | https://raw.githubusercontent.com/appwrite/specs/8c00ba356b01d43fbc6c88c60fb2a4fc99e04b40/specs/1.9.x/open-api3-1.9.x-server.json (and `7d74bf9f3c0c990745e822f5935bcfaca310fd06`) | 2026-11-19 | BSD-3-Clause (appwrite/specs) | `"replaceWith": "tablesDB.createTransaction"` (`x-appwrite.deprecated` of `POST /databases/transactions`, flagged in `7d74bf9f` with 13 more) |
| `appwrite-2.4.x-98cc1a25.json`, `appwrite-2.4.x-06e6fafa.json` | https://raw.githubusercontent.com/appwrite/specs/98cc1a25215d3cead3a64f9bd4574a9c5b1ec87b/specs/2.4.x/open-api3-2.4.x.json (and `06e6fafa4d1ae3c815843ff0b02a56e6e5582756`) | 2026-11-19 | BSD-3-Clause (appwrite/specs) | `"server"` in `x-appwrite.platforms` of `POST /account/recovery/otp` and 12 more operations (in `98cc1a25`, none deprecated; off them in `06e6fafa`, on again in 2.5.x the next day): the drop test |
| `appwrite-1.7.x-ac705cd3-server.json`, `appwrite-1.8.x-ac705cd3-server.json` | https://raw.githubusercontent.com/appwrite/specs/ac705cd3e9d5e3a0693b7e40189eb1e66883e341/specs/1.7.x/open-api3-1.7.x-server.json (and `specs/1.8.x/open-api3-1.8.x-server.json` at the same commit) | 2026-11-19 | BSD-3-Clause (appwrite/specs) | `"replaceWith": "account.createMFAAuthenticator"` on `POST /account/mfa/authenticators/{type}`, whose `x-appwrite.methods` are `createMfaAuthenticator` and `createMFAAuthenticator` (one of 21 SDK method renames flagged in 1.8.x) |

The Appwrite copies keep `openapi`, `info`, `servers` and the path items involved, whole and in their original order; `tags`, `components` and `externalDocs` left out. They're re-serialized the way the originals are written (4-space indent, `/` written `\/`, non-ASCII as `\u` escapes), which gives every original line back byte for byte when nothing is left out, so every kept line is unchanged bar a trailing comma. The 1.9.x pair keeps the paths of the 14 operations deprecated in `7d74bf9f`, the 2.4.x pair the paths of the 13 operations that lost the server platform in `06e6fafa`, the 1.7.x and 1.8.x pair the paths of the 21 operations flagged only for an SDK method rename. With the full files the plugin gives the same results: 14 events, none, and none (50, from the rest of 1.8.x's Databases deprecations, with the 21 renames still left out).

## Recorded API responses

The version-listing tests read these, under `versions/`, instead of the network. Each was recorded on 2026-11-20 from the public API named, then trimmed: only the fields the adapter reads are kept, and some lists are cut to a few entries, said per file. Nothing else is changed. Licenses: these are API metadata (tag names, dates, SHAs), not the projects' content.

| File | Recorded from | Trimmed to |
|---|---|---|
| `versions/appwrite-commits.json` | `https://api.github.com/repos/appwrite/specs/commits?path=specs&per_page=100&page=1` | the 40 newest commits; `sha`, `html_url`, the committer date |
