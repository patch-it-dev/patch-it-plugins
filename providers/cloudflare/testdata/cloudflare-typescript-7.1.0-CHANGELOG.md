## 7.1.0 (2026-08-17)

Full Changelog: [v7.0.0...v7.1.0](https://github.com/cloudflare/cloudflare-typescript/compare/v7.0.0...v7.1.0)

---

### Breaking Changes

#### Resource Restructuring

- `cloudforceOne.threatEvents.indicatorTypes` has been renamed to `cloudforceOne.threatEvents.indicators`

#### Removed Methods

- `intel.attackSurfaceReport.issues.dismiss()` -- removed along with `IssueDismissResponse` type
- `moq.relays.tokens.rotate()` -- removed along with `TokenRotateResponse` type

#### Return Type Changes

| Resource | Method | Old Return Type | New Return Type |
|----------|--------|----------------|-----------------|
| `zones.ct.alerting` | `edit()` | `AlertingEditResponse` | `CTAlertingSubscription` |
| `zones.ct.alerting` | `get()` | `AlertingGetResponse` | `CTAlertingSubscription` |
| `mtlsCertificates` | `list()` | `MTLSCertificatesSinglePage` | `MTLSCertificateListResponsesSinglePage` |
| `mtlsCertificates` | `delete()` | `MTLSCertificate` | `MTLSCertificateDeleteResponse` |
| `mtlsCertificates` | `get()` | `MTLSCertificate` | `MTLSCertificateGetResponse` |
| `hostnames.settings.tls` | `get()` | `TLSGetResponsesSinglePage` | `Setting` |
| `hyperdrive.configs` | `list()` | `HyperdrivesSinglePage` | `HyperdrivesV4PagePaginationArray` |

#### Removed Types

- `AlertingEditResponse`, `AlertingGetResponse` (on `zones.ct.alerting`) -- replaced by `CTAlertingSubscription`
- `MTLSCertificate`, `MTLSCertificatesSinglePage` (on `mtlsCertificates`) -- replaced by per-method response types
- `Host`, `Status`, `ValidationMethod` (on `ssl.certificatePacks`)
- `CertificateAuthority` (on `acm.totalTLS`)
- `HostnameAssociation` (on `certificateAuthorities.hostnameAssociations`)
- `Status` (on `customCertificates`)
- `TLSGetResponse`, `TLSGetResponsesSinglePage` (on `hostnames.settings.tls`)

#### Endpoint Changes

| Resource | Method | Old Endpoint | New Endpoint |
|----------|--------|-------------|--------------|
| `accounts.subscriptions` | `create()` | `POST /accounts/{account_id}/subscriptions` | `POST /{accounts_or_zones}/{account_or_zone_id}/subscriptions` |
| `accounts.subscriptions` | `get()` | `GET /accounts/{account_id}/subscriptions` | `GET /{accounts_or_zones}/{account_or_zone_id}/subscriptions` |
| `loadBalancers` | `create()`, `update()`, `list()`, `delete()`, `edit()`, `get()` | `/zones/{zone_id}/load_balancers/...` | `/{accounts_or_zones}/{account_or_zone_id}/load_balancers/...` |
| `billing.usage` | `paygo()` | `GET /accounts/{account_id}/paygo-usage` | `GET /accounts/{account_id}/billable-usage` |
| `hostnames.settings.tls` | `get()` | `GET /zones/{zone_id}/hostnames/settings/{setting_id}` | `GET /zones/{zone_id}/hostnames/settings/{setting_id}/{hostname}` |

---

### Features

#### New Resources

- **`analyticsQuery`** -- `summary()`, `timeseries()`, `topN()` for analytics queries
- **`analyticsQuery.dataSecurity.contentFindings`** -- `topN()`
- **`analyticsQuery.dataSecurity.findings`** -- `summary()`, `timeseries()`
- **`accounts.speedSettings.transformations`** -- `get()`
- **`zones.transformationsAllowedOrigins`** -- `edit()`, `get()`
- **`zones.transformationsC2pa`** -- `edit()`, `get()`
- **`customHostnames.quota`** -- `get()`
- **`dns.dnssec.zsk`** -- `list()`
- **`emailRouting.accountRules`** -- `list()`
- **`intel.urls`** -- `get()`
- **`intel.sinkholes.ingresses`** -- `create()`, `update()`, `delete()`, `get()`
- **`magicTransit.connectors.interrupts`** -- `create()`, `list()`
- **`pages.projects.deployments.tails`** -- `create()`, `delete()`
- **`pages.assets`** -- `checkMissing()`, `upload()`, `upsertHashes()`
- **`precursor`** -- `update()`, `get()`
- **`registrar.extensions`** -- `list()`, `get()`
- **`registrarSandbox`** -- `check()`, `search()` with sub-resources for `registrations`, `registrationStatus`, `updateStatus`, and `extensions`
- **`cloudforceOne.threatEvents.aggregate`** -- `list()`
- **`cloudforceOne.threatEvents.graphql`** -- `create()`
- **`cloudforceOne.threatEvents.graph`** -- `list()`
- **`cloudforceOne.threatEvents.queries`** -- `create()`, `list()`, `delete()`, `edit()`, `get()`
- **`cloudforceOne.threatEvents.relationships`** -- `list()`
- **`cloudforceOne.threatEvents.indicators`** -- with sub-resources for `aggregate`, `types`, `byDataset`, and `byDataset.tags`
- **`cloudforceOne.threatEvents.categories.catalog`** -- `list()`
- **`cloudforceOne.threatEvents.datasets.events`** -- `get()`
- **`cloudforceOne.threatEvents.tags`** -- with sub-resources for `categories`, `indicators`, and `indicators.byDataset`
- **`cloudforceOne.threatEvents.targetIndustries`** -- with sub-resources for `byDataset` and `catalog`
- **`radar.bgp.routes.upstreams`** -- `timeseries()`
- **`radar.bgp.routes.paths`** -- `list()`
- **`zeroTrust.casb.applications.authMethods`** -- `list()`
- **`zeroTrust.casb.posture`** -- with sub-resources for `findings`, `exports`, `findingTypes`, `findingTypes.remediationTypes`, `content`, `remediations.jobs`, `webhooks`, and `webhooks.jobs`
- **`zeroTrust.networks.subnets.initialResolvedIP`** -- `update()`, `get()`

#### New Methods on Existing Resources

- **`emailSecurity.settings.domains.bulkDelete()`** -- `DELETE /accounts/{account_id}/email-security/settings/domains`
- **`emailRouting.update()`** -- `PUT /zones/{zone_id}/email/routing`
- **`emailRouting.edit()`** -- `PATCH /zones/{zone_id}/email/routing`
- **`emailRouting.rules.list()`** -- `GET /{accounts_or_zones}/{account_or_zone_id}/email/routing/rules`
- **`logs.logExplorer.datasets.delete()`** -- `DELETE /{accounts_or_zones}/{account_or_zone_id}/logs/explorer/datasets/{dataset_id}`
- **`queues.messages.peek()`** -- `POST /accounts/{account_id}/queues/{queue_id}/messages/peek`
- **`queues.messages.purge()`** -- `POST /accounts/{account_id}/queues/{queue_id}/messages/purge`
- **`addressing.prefixes.validate()`** -- `POST /accounts/{account_id}/addressing/prefixes/{prefix_id}/validate`
- **`billing.usage.getAccountUsageInfoV1()`** -- `GET /accounts/{account_id}/billable-usage/info`
- **`billing.usage.getAccountUsageV1()`** -- `GET /accounts/{account_id}/billable-usage`
- **`billing.usage.getAccountUsageV2()`** -- `GET /accounts/{account_id}/billable/usage`
- **`billing.usage.paygoInfo()`** -- `GET /accounts/{account_id}/billable-usage/info`
- **`intel.sinkholes.create()`**, **`update()`**, **`delete()`**, **`get()`** -- CRUD for sinkholes
- **`pages.projects.getUploadToken()`** -- `GET /accounts/{account_id}/pages/projects/{project_name}/upload-token`
- **`zeroTrust.casb.applications.list()`**, **`get()`** -- list and get CASB applications
- **`zeroTrust.casb.integrations`** -- `create()`, `update()`, `list()`, `delete()`, `get()`, `pause()`, `resume()`
- **`hostnames.settings.tls.list()`** -- `GET /zones/{zone_id}/hostnames/settings/{setting_id}`
- **`moq.relays.tokens.create()`**, **`list()`**, **`delete()`** -- token management
- **`cloudforceOne.threatEvents.bulkCreateRelationships()`** -- bulk create relationships
- **`cloudforceOne.threatEvents.datasets.delete()`** -- delete datasets
- **`cloudforceOne.threatEvents.tags.list()`**, **`delete()`**, **`edit()`** -- tag management
- **`resourceTagging.summary.get()`** -- `GET /accounts/{account_id}/tags/summary`
- **`aiAudit.robots.bulkGet()`**, **`get()`** -- AI audit robot data

---

### Chores

- Updated `@arethetypeswrong/cli` from ^0.17.0 to ^0.18.0
- Fixed `repository` field in `package.json`

## 7.0.0 (2026-07-09)

Full Changelog: [v6.5.0...v7.0.0](https://github.com/cloudflare/cloudflare-typescript/compare/v6.5.0...v7.0.0)

This is a major release with significant breaking changes to the SDK internals and API surface.
The SDK now uses the builtin Web fetch API, has zero runtime dependencies, and adopts a new
parameter convention for methods with multiple path parameters. A migration CLI is included to
help automate code updates.

Please read through the breaking changes below and the [Migration Guide](./MIGRATION.md) before
upgrading. You can also run the automated migration tool:

```sh
npx cloudflare migrate ./your/src/folders
```

#### Known limitations of the migration tool

- **Not idempotent** -- running it a second time on already-migrated code will produce
  incorrect output. Run it once only.
- **Partial migrations** -- if you have manually updated some call sites to v7 style
  before running the tool, those call sites may be incorrectly transformed. Review
  the output diff carefully before committing.

---

### General Changes

#### Zero dependencies

The SDK no longer depends on `node-fetch`, `agentkeepalive`, `form-data-encoder`, `formdata-node`,
or `abort-controller`. It uses the builtin Web fetch API on all platforms. The `dependencies` field
in `package.json` is now empty.

#### New client options

- **`fetchOptions`** -- pass additional `RequestInit` options to every `fetch` call (e.g. custom
  `signal`, `keepalive`, `cache`). Per-request `fetchOptions` override the client-level default.
- **`logLevel`** / **`logger`** -- configure SDK logging. Defaults to `process.env['CLOUDFLARE_LOG']`
  or `'warn'`. Set `logLevel: 'debug'` for request/response tracing.

#### Removed client options

- **`httpAgent`** -- removed. Use `fetchOptions` to pass agent-like configuration instead.

#### Type changes

- `defaultHeaders` type changed from `Core.Headers` to `HeadersLike`
- `fetch` type changed from `Core.Fetch` to `Fetch`

#### Internal restructuring

The SDK internals have been reorganized:
- `src/core.ts` -> split into `src/core/` modules (`api-promise.ts`, `error.ts`, `pagination.ts`,
  `resource.ts`, `uploads.ts`)
- `src/_shims/` -> replaced by `src/internal/` (`builtin-types.ts`, `detect-platform.ts`,
  `errors.ts`, `headers.ts`, `parse.ts`, `shims.ts`, etc.)
- `src/index.ts` now re-exports from `src/client.ts`

Import paths for internal modules have changed. If you were importing from `cloudflare/core` or
`cloudflare/_shims`, update your imports accordingly.

---

### Breaking Changes

#### Named path parameters

Methods that take multiple path parameters now use named instead of positional arguments for all
but the _last_ path parameter. This prevents a common footgun where arguments could be accidentally
passed in the wrong order.

```ts
// Before (v6.x)
client.accounts.members.get(memberId, { account_id: '...' });

// After (v7.0.0)
client.accounts.members.get(memberID, { account_id: '...' });
```

For methods with 3+ path parameters, intermediate parameters move into the options object:

```ts
// Before (v6.x)
client.customHostnames.certificatePack.certificates.update(
  customHostnameId, certificatePackId, certificateId, { ...params }
);

// After (v7.0.0)
client.customHostnames.certificatePack.certificates.update(
  certificateID, { custom_hostname_id: '...', certificate_pack_id: '...', ...params }
);
```

This affects 232 methods across the SDK. See the [Migration Guide](./MIGRATION.md) for the
complete list.

#### Parameter name casing (`camelCase` -> `UPPER_ID`)

Path parameter argument names have been standardized from `camelCase` to `UPPER_ID` format
across 792 methods:

```ts
// Before (v6.x)
client.alerting.policies.get(policyId, { ...params });

// After (v7.0.0)
client.alerting.policies.get(policyID, { ...params });
```

While this is not a runtime breaking change (positional arguments work regardless of name),
it is a TypeScript compilation change if you were using named/destructured parameters.

#### Web types for `withResponse`, `asResponse`, and `APIError.headers`

The SDK now uses the builtin Web fetch API on all platforms. If you accessed `node-fetch`-specific
properties on response objects, you need to switch to standardized alternatives:

```ts
// Before (v6.x)
const res = await client.example.retrieve('id').asResponse();
res.body.pipe(process.stdout);

// After (v7.0.0)
import { Readable } from 'node:stream';
const res = await client.example.retrieve('id').asResponse();
Readable.fromWeb(res.body).pipe(process.stdout);
```

The `headers` property on `APIError` objects is now an instance of the Web `Headers` class
(previously `Record<string, string | null | undefined>`).

#### Return type changes

| Resource | Method | Old Return Type | New Return Type |
|----------|--------|----------------|-----------------|
| `zeroTrust.devices.dexTests` | `create()` | `DEXTestCreateResponse` | `SchemaHTTP` |
| `zeroTrust.devices.dexTests` | `update()` | `DEXTestUpdateResponse` | `SchemaHTTP` |
| `zeroTrust.devices.dexTests` | `list()` | `DEXTestListResponsesV4PagePaginationArray` | `SchemaHTTPSV4PagePaginationArray` |
| `zeroTrust.devices.dexTests` | `get()` | `DEXTestGetResponse` | `SchemaHTTP` |

#### Removed types

- `DEXTestCreateResponse` (use `SchemaHTTP`)
- `DEXTestUpdateResponse` (use `SchemaHTTP`)
- `DEXTestListResponse` (use `SchemaHTTPS`)
- `DEXTestGetResponse` (use `SchemaHTTP`)

#### Method renames (`Id` -> `ID`)

Several methods were renamed to use consistent `ID` casing:

| Resource | Old Method | New Method |
|----------|-----------|------------|
| `vectorize.indexes` | `deleteByIds()` | `deleteByIDs()` |
| `vectorize.indexes` | `getByIds()` | `getByIDs()` |
| `realtimeKit.meetings` | `getMeetingById()` | `getMeetingByID()` |
| `realtimeKit.meetings` | `replaceMeetingById()` | `replaceMeetingByID()` |
| `realtimeKit.meetings` | `updateMeetingById()` | `updateMeetingByID()` |
| `realtimeKit.presets` | `getPresetById()` | `getPresetByID()` |
| `realtimeKit.sessions` | `getParticipantDataFromPeerId()` | `getParticipantDataFromPeerID()` |
| `realtimeKit.webhooks` | `getWebhookById()` | `getWebhookByID()` |
| `realtimeKit.livestreams` | `getActiveLivestreamsForLivestreamId()` | `getActiveLivestreamsForLivestreamID()` |
| `realtimeKit.livestreams` | `getLivestreamSessionDetailsForSessionId()` | `getLivestreamSessionDetailsForSessionID()` |
| `realtimeKit.livestreams` | `getLivestreamSessionForLivestreamId()` | `getLivestreamSessionForLivestreamID()` |

#### Removed `fileFromPath` helper

The `Cloudflare.fileFromPath` static method and the named `fileFromPath` export have been removed.
Use native Node.js streams or the `toFile()` helper instead:

```ts
// Before (v6.x)
import Cloudflare, { fileFromPath } from 'cloudflare';
const file = await fileFromPath('path/to/file');

// After (v7.0.0)
import fs from 'node:fs';
const file = fs.createReadStream('path/to/file');
```

#### `APIClient` replaced by `BaseCloudflare`

The `APIClient` base class has been replaced:

```ts
// Before (v6.x)
import { APIClient } from 'cloudflare/core';

// After (v7.0.0)
import { BaseCloudflare } from 'cloudflare/client';
```

#### Page classes converted to type aliases

Per-method pagination classes (e.g. `AccountsV4PagePaginationArray`) are now type aliases instead
of classes. Runtime `instanceof` checks against them will break. Import the base pagination class
instead, or use them only at the type level.

#### `cloudflare/src` directory removed

The `cloudflare/src/*` import paths are no longer available. If your IDE auto-completed imports
from `cloudflare/src/...`, replace them with `cloudflare/...`.

---

### Features

#### New Resources

- **`registrar.registrations`** -- `create()`, `list()`, `edit()`, `get()` for domain registrations

#### New Methods

- **`accounts.logs.audit.history()`** -- `GET /accounts/{account_id}/logs/audit/{id}/history`
- **`accounts.logs.audit.productCategories()`** -- `GET /accounts/{account_id}/logs/audit/product_categories`
- **`organizations.logs.audit.history()`** -- `GET /organizations/{organization_id}/logs/audit/{id}/history`
- **`emailRouting.unlock()`** -- `POST /zones/{zone_id}/email/routing/unlock`
- **`emailRouting.addresses.edit()`** -- `PATCH /accounts/{account_id}/email/routing/addresses/{destination_address_identifier}`
- **`browserRendering.accessibilityTree.create()`** -- `POST /accounts/{account_id}/browser-rendering/accessibilityTree`

#### New Types

- `AuditHistoryResponse` (on `accounts.logs.audit` and `organizations.logs.audit`)
- `AuditProductCategoriesResponse` (on `accounts.logs.audit`)

#### Tree-shaking support

New `createClient` function and `PartialCloudflare` type exported from `cloudflare/tree-shakable`.
Every resource class now exports a `Base*` variant (e.g. `BaseAccounts`, `BaseZones`) for selective
imports, allowing bundlers to tree-shake unused resources and reduce bundle size.

```ts
import { createClient } from 'cloudflare/tree-shakable';
import { Zones } from 'cloudflare/resources/zones/zones';

const client = createClient({ resources: [Zones] });
```

#### Migration CLI

Automated codemod for v6 -> v7 migration. Handles path parameter reordering, casing changes,
and import path updates:

```sh
npx cloudflare migrate ./your/src/folders
```

---

### Bug Fixes

- **workflows:** move Workflow payload types into `Payload` namespace to avoid TS2717 duplicate
  identifier errors

---

### Chores

- Zero runtime dependencies (removed `node-fetch`, `agentkeepalive`, `form-data-encoder`,
  `formdata-node`, `abort-controller`)
- TypeScript upgraded to 5.8.3 (from ^4.8.2)
- ESLint upgraded to ^9.39.1 (from ^8.49.0)
- Node.js minimum version: 20 LTS
- Internal module restructuring (`src/core.ts` -> `src/core/`, `src/_shims/` -> `src/internal/`)
