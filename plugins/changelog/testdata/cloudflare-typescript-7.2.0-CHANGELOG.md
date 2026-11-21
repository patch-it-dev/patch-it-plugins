## 7.2.0 (2026-09-25)

Full Changelog: [v7.1.0...v7.2.0](https://github.com/cloudflare/cloudflare-typescript/compare/v7.1.0...v7.2.0)

---

### Breaking Changes

#### Removed Methods

- `realtimeKit.livestreams.createIndependentLivestream()` -- removed along with `LivestreamCreateIndependentLivestreamResponse` type

#### End-of-life APIs

The legacy Rate Limiting and WAF overrides APIs have reached end of life and now respond with `410 Gone`. All methods
on these resources now return `void`, and their model types have been removed.

| Resource | Methods | Removed Types |
|----------|---------|---------------|
| `rateLimits` | `create()`, `list()`, `delete()`, `edit()`, `get()` | `RateLimit`, `RateLimitsV4PagePaginationArray`, `RateLimitDeleteResponse` |
| `firewall.waf.overrides` | `create()`, `update()`, `list()`, `delete()`, `get()` | `Override`, `OverridesV4PagePaginationArray`, `OverrideDeleteResponse` |

Migrate to the [Rulesets API](https://developers.cloudflare.com/ruleset-engine/) (`rulesets`) for rate limiting and
managed ruleset overrides.

#### R2 `jurisdiction` parameter renamed

The `jurisdiction` header parameter on all `r2.buckets` methods (including `cors`, `domains.custom`,
`domains.managed`, `eventNotifications`, `lifecycle`, `locks`, `objects`, and `sippy`) is now named
`'cf-r2-jurisdiction'`. The allowed values now also include `'us'` and `'fedramp-high'`.

```ts
// Before
await client.r2.buckets.get('my-bucket', { account_id, jurisdiction: 'eu' });

// After
await client.r2.buckets.get('my-bucket', { account_id, 'cf-r2-jurisdiction': 'eu' });
```

#### `addressing.addressMaps.accounts` path parameters

`update()` and `delete()` now take the member account ID as the positional argument, and `address_map_id` is a
required param. The unused `body` param on `update()` has been removed. The endpoint is now
`/accounts/{account_id}/addressing/address_maps/{address_map_id}/accounts/{member_account_id}`.

```ts
// Before
await client.addressing.addressMaps.accounts.update(addressMapID, { account_id, body: {} });
await client.addressing.addressMaps.accounts.delete(addressMapID, { account_id });

// After
await client.addressing.addressMaps.accounts.update(memberAccountID, { account_id, address_map_id: addressMapID });
await client.addressing.addressMaps.accounts.delete(memberAccountID, { account_id, address_map_id: addressMapID });
```

The `cloudflare migrate` tool has been updated to rewrite these call sites.

#### `firewall.rules` bulk methods

The legacy Firewall Rules API is deprecated and returns `410 Gone`; use the
[Rulesets API](https://developers.cloudflare.com/ruleset-engine/) (`rulesets`) instead. To match the published schema,
`bulkEdit()` and `bulkUpdate()` no longer accept a `body` param and now require an `id` param.

```ts
// Before
await client.firewall.rules.bulkEdit({ zone_id, body: [...] });

// After
await client.firewall.rules.bulkEdit({ zone_id, id: ruleID });
```

#### Empty `body` params removed

The following methods no longer accept a placeholder `body` param. Remove `body: {}` from call sites.

- `accounts.tokens.value.update()`
- `addressing.addressMaps.ips.update()`, `addressing.addressMaps.zones.update()`
- `cache.cacheReserve.clear()`
- `dns.records.scan()`
- `dns.zoneTransfers.forceAXFR.create()`
- `dns.zoneTransfers.outgoing.disable()`, `enable()`, `forceNotify()`
- `emailRouting.disable()`, `emailRouting.enable()`
- `magicTransit.ipsecTunnels.pskGenerate()`
- `stream.keys.create()`
- `workers.scripts.tail.create()`
- `zeroTrust.access.bookmarks.create()`, `update()`
- `zeroTrust.gateway.certificates.activate()`, `deactivate()`

```ts
// Before
await client.dns.records.scan({ zone_id, body: {} });

// After
await client.dns.records.scan({ zone_id });
```

#### Other parameter changes

- `browserRendering` -- params for `content`, `json`, `links`, `markdown`, `pdf`, `scrape`, `screenshot`,
  `snapshot`, and `accessibilityTree` are now a union of `Variant0` (requires `url`) and `Variant1` (requires
  `html`). Code that declares these param types explicitly must provide exactly one of `url` or `html`.
- `magicTransit.connectors` -- `primary` and `site_id` removed from `create()`, `update()`, and `edit()` params and
  from all responses
- `cloudforceOne.threatEvents.tags` -- `actorCategoryConfidence`, `analyticPriority`, `attributionConfidence`,
  `attributionConfidenceScore`, `motiveConfidence`, `originCountryConfidence`, and `originCountryTlp` removed from
  `create()` and `edit()` params; these fields and `originCountryISOAlpha3` removed from responses
- `pipelines.sinks`, `pipelines.streams` -- `schema.format` removed from `create()` params and responses; `force`
  removed from `delete()` params
- `workflows.instances.events.create()`, `workflows.instances.bulk()` -- `body` is now required

#### Return Type Changes

| Resource | Method | Old Return Type | New Return Type |
|----------|--------|----------------|-----------------|
| `pageShield` | `get()` | `Setting \| null` | `PageShieldGetResponse \| null` |
| `pageShield.connections` | `list()` | `ConnectionsSinglePage` | `ConnectionListResponsesSinglePage` |
| `pageShield.connections` | `get()` | `Connection \| null` | `ConnectionGetResponse \| null` |
| `pageShield.scripts` | `list()` | `ScriptsSinglePage` | `ScriptListResponsesSinglePage` |
| `schemaValidation.schemas` | `delete()` | `SchemaDeleteResponse` | `SchemaDeleteResponse \| null` |
| `cache.originCloudRegions` | `update()` | `OriginCloudRegion` | `OriginCloudRegionUpdateResponse` |
| `cache.originCloudRegions` | `list()` | `OriginCloudRegionsV4PagePaginationArray` | `OriginCloudRegionListResponsesV4PagePaginationArray` |
| `cache.originCloudRegions` | `get()` | `OriginCloudRegion` | `OriginCloudRegionGetResponse` |
| `hyperdrive.configs` | `create()` | `Hyperdrive` | `ConfigCreateResponse` |
| `hyperdrive.configs` | `update()` | `Hyperdrive` | `ConfigUpdateResponse` |
| `hyperdrive.configs` | `list()` | `HyperdrivesV4PagePaginationArray` | `ConfigListResponsesV4PagePaginationArray` |
| `hyperdrive.configs` | `edit()` | `Hyperdrive` | `ConfigEditResponse` |
| `hyperdrive.configs` | `get()` | `Hyperdrive` | `ConfigGetResponse` |
| `mtlsCertificates` | `list()` | `MTLSCertificateListResponsesSinglePage` | `MTLSCertificatesSinglePage` |
| `mtlsCertificates` | `delete()` | `MTLSCertificateDeleteResponse` | `MTLSCertificate` |
| `mtlsCertificates` | `get()` | `MTLSCertificateGetResponse` | `MTLSCertificate` |
| `snippets.rules` | `update()` | `RuleUpdateResponse` | `RuleUpdateResponsesSinglePage` |
| `snippets.rules` | `list()` | `RuleListResponse` | `RuleListResponsesSinglePage` |
| `snippets.rules` | `delete()` | `RuleDeleteResponse` | `RuleDeleteResponsesSinglePage` |
| `snippets.rules` | `get()` | `RuleGetResponse` | `RuleGetResponsesSinglePage` |
| `workers.scripts.deployments` | `list()` | `DeploymentListResponse` | `DeploymentListResponsesV4PagePagination` |
| `zeroTrust.devices.overrideCodes` | `list()` | `OverrideCodeListResponsesSinglePage` | `OverrideCodeListResponse \| null` |
| `zeroTrust.organizations` | `list()` | `Organization` | `OrganizationListResponse` |
| `zeroTrust.gateway.lists.items` | `list()` | `GatewayItemsSinglePage` | `GatewayItemsV4PagePaginationArray` |
| `zeroTrust.gateway.pacfiles` | `list()` | `PacfileListResponsesSinglePage` | `PacfileListResponsesV4PagePaginationArray` |
| `emailRouting.dns` | `delete()` | `DNSRecordsSinglePage` | `Settings` |
| `waitingRooms` | `create()`, `update()`, `edit()`, `get()` | `WaitingRoom` | `WaitingRoom \| null` |
| `waitingRooms` | `delete()` | `WaitingRoomDeleteResponse` | `WaitingRoomDeleteResponse \| null` |
| `waitingRooms.page` | `preview()` | `PagePreviewResponse` | `PagePreviewResponse \| null` |
| `waitingRooms.events` | `create()`, `update()`, `edit()`, `get()` | `Event` | `Event \| null` |
| `waitingRooms.events` | `delete()` | `EventDeleteResponse` | `EventDeleteResponse \| null` |
| `waitingRooms.events.details` | `get()` | `DetailGetResponse` | `DetailGetResponse \| null` |
| `waitingRooms.statuses` | `get()` | `StatusGetResponse` | `StatusGetResponse \| null` |
| `waitingRooms.settings` | `update()` | `SettingUpdateResponse` | `SettingUpdateResponse \| null` |
| `waitingRooms.settings` | `edit()` | `SettingEditResponse` | `SettingEditResponse \| null` |
| `waitingRooms.settings` | `get()` | `SettingGetResponse` | `SettingGetResponse \| null` |

Paginated `list()` methods that moved from `SinglePage` to `V4PagePagination*` now auto-paginate across pages when
iterated with `for await`. The `mtlsCertificates` change reverts the per-method response types introduced in 7.1.0.

#### Removed Types

- `Setting` (on `pageShield`) -- replaced by `PageShieldUpdateResponse` (for `edit()`) and `PageShieldGetResponse` (for `get()`)
- `Policy` (on `pageShield.policies`) -- replaced by per-method response types
- `Connection`, `ConnectionsSinglePage` (on `pageShield.connections`) -- replaced by `ConnectionListResponse`, `ConnectionListResponsesSinglePage`
- `Script`, `ScriptsSinglePage` (on `pageShield.scripts`) -- replaced by `ScriptListResponse`, `ScriptListResponsesSinglePage`
- `LivestreamCreateIndependentLivestreamResponse` (on `realtimeKit.livestreams`) -- removed with method
- `SchemaDeleteResponse` changed from `interface { id: string }` to `type = unknown` -- users accessing `.id` on the result will see a type error
- `HealthCheckParam` (on `magicTransit`) -- removed; `health_check` params now reference types scoped to each interconnect method (e.g. `CfInterconnectUpdateParams.HealthCheck`)
- 72 `WorkersBindingKind*` types removed from `workers.scripts.versions.VersionListResponse` -- the `bindings` field in `VersionListResponse.Resources` is now typed as an empty `Bindings` interface rather than an explicit union of binding variants
- `OriginCloudRegion`, `OriginCloudRegionsV4PagePaginationArray` (on `cache.originCloudRegions`) -- replaced by per-method response types
- `Configuration`, `HyperdrivesV4PagePaginationArray` (on `hyperdrive`) -- replaced by `ConfigCreateResponse`, `ConfigListResponse`, etc.
- `MTLSCertificateListResponse`, `MTLSCertificateDeleteResponse`, `MTLSCertificateGetResponse`, `MTLSCertificateListResponsesSinglePage` (on `mtlsCertificates`) -- replaced by `MTLSCertificate`
- `ClipResource`, `BaseClipResource` (on `stream`) -- renamed to `Clip`, `BaseClip`; the former `Clip` model interface has been removed (`stream.clip.create()` returns `Video`)
- `State` (on `cache.cacheReserve`)
- `CacheVariant` (on `cache.variants`)
- `Aegis` (on `zones.settings`)
- `Info`, `Submit` (on `brandProtection`)
- `Whois` (on `intel.whois`)
- `FallbackDomainPolicy` (on `zeroTrust.devices.policies`)
- `Connection` (on `zeroTrust.tunnels`)
- `OverrideCodeListResponsesSinglePage` (on `zeroTrust.devices.overrideCodes`)
- `GatewayItemsSinglePage` (on `zeroTrust.gateway.lists`) -- replaced by `GatewayItemsV4PagePaginationArray`
- `PacfileListResponsesSinglePage` (on `zeroTrust.gateway.pacfiles`) -- replaced by `PacfileListResponsesV4PagePaginationArray`

#### Removed Response Fields

- `apiGateway.operations`, `apiGateway.userSchemas.operations` -- `features.schema_info.learned_available` and
  `features.schema_info.active_schema.is_learned`
- `radar.annotations.list()`, `radar.annotations.outages.get()` -- `asnsDetails[].locations`
- `radar.trafficAnomalies.get()` -- `asnDetails.locations`
- `schemaValidation.settings.operations.bulkEdit()` -- `operation_id` on each item
- `stream.webhooks.update()`, `stream.webhooks.get()` -- `notification_url`
- `tokenValidation.configuration.credentials.update()` -- `errors`, `messages`, `success` envelope fields
- `urlScanner.scans.get()` -- `meta.processors.robotsTxt.data[].rules['*']`
- `user.billing.profile.get()` -- `device_data`, `payment_gateway`, `payment_nonce`, `use_legacy`
- `zones.plans` (`AvailableRatePlan`) -- `legacy_discount`

---

### Features

#### New Resources

- **`managedDefense.vulnerabilityDiscovery.repositories`** -- `create()`, `list()`, `get()`
- **`managedDefense.vulnerabilityDiscovery.scans`** -- `create()`, `list()`, `get()`, `getReport()`
- **`managedDefense.vulnerabilityDiscovery.reports`** -- `get()`
- **`fieldExtractors`** -- `update()`, `delete()`, `get()`
- **`accounts.subscriptions.cancelReason`** -- `create()`, `get()`
- **`accounts.subscriptions.actions`** -- `append()`
- **`accounts.subscriptions.bulk`** -- `create()`
- **`accounts.entitlements`** -- `list()`
- **`accounts.paymentMethods`** -- `create()`, `update()`, `list()`, `delete()`, `get()`, `setAsDefault()`
- **`accounts.payInvoice`** -- `create()`
- **`accounts.payBadDebt`** -- `create()`
- **`accounts.receipts`** -- `pdf()`
- **`accounts.invoices`** -- `edit()`
- **`accounts.clientSecret`** -- `create()`
- **`billing.profiles.paymentMethod`** -- `create()`
- **`billing.credits`** -- `get()`
- **`billing.history`** -- `list()`
- **`billing.badDebt`** -- `get()`
- **`billing.unpaidInvoice`** -- `get()`
- **`billing.ratePlans`** -- `get()`
- **`user.spectrumAnalytics.zones.reports`** -- `get()`
- **`zones.nel`** -- `edit()`, `get()`
- **`zones.entitlements`** -- `list()`
- **`zones.observability.tracing.settings`** -- `update()`, `delete()`, `get()`
- **`zones.observability.tracing.rules`** -- `update()`, `delete()`, `get()`
- **`emailSecurity.settings.contentPolicies`** -- `create()`, `list()`, `delete()`, `batch()`, `edit()`, `get()`
- **`emailSending.suppressions`** -- `create()`, `list()`, `delete()`, `edit()`, `get()`, `import()`
- **`logpush.transformers`** -- `create()`, `update()`, `list()`, `delete()`, `get()`, `preview()`
- **`logpush.transformers.content`** -- `get()`
- **`logpush.transformers.versions`** -- `list()`
- **`spectrum.protocols`** -- `list()`
- **`magicTransit.bgpFilterProfiles`** -- `create()`, `update()`, `list()`, `delete()`, `get()`
- **`registrar.transferIn`** -- `create()`
- **`registrar.transferInStatus`** -- `get()`
- **`zeroTrust.casb.posture.policies`** -- `create()`, `update()`, `list()`, `delete()`, `get()`
- **`abuseReports.submitted`** -- `list()`, `get()`
- **`abuseReports.submitted.emails`** -- `list()`
- **`browserRendering.devtools.browser.liveView`** -- `create()`

#### New Methods

- `accounts.subscriptions.cancelDowngrade()`, `getByIdentifier()`
- `billing.addressValidation()`
- `billing.profiles.create()`, `update()`, `delete()`, `updateBillingEmail()`
- `ssl.recommendations.get()`
- `emailSecurity.settings.allowPolicies.batch()`
- `emailSecurity.settings.blockSenders.batch()`
- `emailSecurity.settings.domains.create()`, `update()`, `batch()`
- `emailSecurity.settings.trustedDomains.batch()`
- `emailSending.subdomains.edit()`
- `addressing.prefixes.bgpPrefixes.delete()`
- `hyperdrive.configs.restart()`
- `r2DataCatalog.delete()`
- `zeroTrust.resourceLibrary.applications.create()`, `update()`, `delete()`
- `realtimeKit.presets.replacePresetByID()`
- `contentScanning.payloads.update()`
- `tokenValidation.configuration.credentials.edit()`

#### New Types

- `Host`, `Status`, `ValidationMethod` (on `ssl.certificatePacks`) and `CertificateAuthority` (on `acm.totalTLS`) -- restored
- `FieldType`, `ListField`, `SourceField`, `StructField` (on `pipelines`)
- `cloudforceOne.threatEvents.tags` -- annotated value types for tag attribution and confidence fields

---

### Chores

- Updated the `cloudflare migrate` tool configuration for `addressing.addressMaps.accounts` and new methods
- Updated the method lists in `MIGRATION.md`

---

## 6.5.0 (2026-06-23)

Full Changelog: [v6.3.0...v6.5.0](https://github.com/cloudflare/cloudflare-typescript/compare/v6.3.0...v6.5.0)

### Features

* **ai-gateway:** add CustomProviders, 1 sub-resource ([456be6f](https://github.com/cloudflare/cloudflare-typescript/commit/456be6ff8e4a85de321f6ac2dd466e0d572f9e85))
* **email-auth:** add EmailAuth resource ([f424bee](https://github.com/cloudflare/cloudflare-typescript/commit/f424bee5791d76683534ba0721a1e79cba1c1954))
* **email-security:** add bulk investigation ([827bf4c](https://github.com/cloudflare/cloudflare-typescript/commit/827bf4c8a36b90217765eaed56b47206914d3ea5))
* **logs:** add log-explorer sub-resource ([5125978](https://github.com/cloudflare/cloudflare-typescript/commit/512597855200c2dd8015ae482061b6f0f9fbb2da))
* **magic-transit:** add cf1-sites sub-resource ([148612a](https://github.com/cloudflare/cloudflare-typescript/commit/148612a4e52573e0a2500fb06b38e352515206fc))
* **moq:** add MoQ resource ([2712713](https://github.com/cloudflare/cloudflare-typescript/commit/2712713470b5867c3cc8c6dc3371adf8e3656a1c))
* **tenants:** add Tenants resource ([035563c](https://github.com/cloudflare/cloudflare-typescript/commit/035563c1d81f5a1eb952179d4faf449191fe4732))
* **user:** add user tenants sub-resource ([803425f](https://github.com/cloudflare/cloudflare-typescript/commit/803425ff57f5195069325a02c967beefd56c5058))
* **zones:** add CT alerting sub-resource ([9a3dead](https://github.com/cloudflare/cloudflare-typescript/commit/9a3dead21a27e49473e8dccd21c066e5697ceef0))


### Chores

* **accounts:** update codegen output ([9f670fd](https://github.com/cloudflare/cloudflare-typescript/commit/9f670fd360d4a6248fc265c45040df86fccf4f0f))
* **aisearch:** update codegen output ([4d687e0](https://github.com/cloudflare/cloudflare-typescript/commit/4d687e0c8d179352c0473ec24fb451930c1b9614))
* **billing:** update codegen output ([db0b645](https://github.com/cloudflare/cloudflare-typescript/commit/db0b6457c57776ffcf65124e60483e219235a211))
* **browser-rendering:** update codegen output ([548a32f](https://github.com/cloudflare/cloudflare-typescript/commit/548a32f1add9bac6e3fa99cfe8af2c89173345f9))
* **cloudforce-one:** update codegen output ([e2aa54a](https://github.com/cloudflare/cloudflare-typescript/commit/e2aa54a9b7fc433221cbc57445a76590b25f81fd))
* **custom-hostnames:** update codegen output ([cc890d6](https://github.com/cloudflare/cloudflare-typescript/commit/cc890d67e1fd26348d6c40afb5f5e42f1423b880))
* **email-routing:** update codegen output ([90eea55](https://github.com/cloudflare/cloudflare-typescript/commit/90eea555f6e84cbc4002704af4b8e1645b842acb))
* **iam:** update codegen output ([ed876e6](https://github.com/cloudflare/cloudflare-typescript/commit/ed876e69b255c654646547704bb68d46793cae18))
* **intel:** update codegen output ([271795f](https://github.com/cloudflare/cloudflare-typescript/commit/271795fd9666ca952d8c48763682f8d509d544a4))
* **origin-tls-compliance-modes:** update codegen output ([704cd50](https://github.com/cloudflare/cloudflare-typescript/commit/704cd50a4a504842fcf4d04ee8959b53b8b135e3))
* **radar:** update codegen output ([8c02e68](https://github.com/cloudflare/cloudflare-typescript/commit/8c02e6818903b5afe4d0dd84d622a7473adcb331))
* **rulesets:** update codegen output ([edf0d01](https://github.com/cloudflare/cloudflare-typescript/commit/edf0d01dd93e2e64cd73f13732e89be1403f51ca))
* **ssl:** update codegen output ([667b5b3](https://github.com/cloudflare/cloudflare-typescript/commit/667b5b3cda91552db9614fe9d4af2e9442028d25))
* sync shared codegen files from staging-next ([6098a42](https://github.com/cloudflare/cloudflare-typescript/commit/6098a42d775eed6d5c446fa7537a2d5bd640e5d2))
* **workers-for-platforms:** update codegen output ([ef238e6](https://github.com/cloudflare/cloudflare-typescript/commit/ef238e61422201a42dbbf4094037bcaed85a80a9))
* **workers:** update codegen output ([51222e9](https://github.com/cloudflare/cloudflare-typescript/commit/51222e95d26965e02df3cb14352849210087cc59))
* **zero-trust:** update codegen output ([aa2cbc3](https://github.com/cloudflare/cloudflare-typescript/commit/aa2cbc34e9348680e304820d8dbfb84d1028de64))
