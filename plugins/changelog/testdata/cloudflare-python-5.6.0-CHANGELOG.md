## 5.6.0 (2026-07-22)

Full Changelog: [v5.5.0...v5.6.0](https://github.com/cloudflare/cloudflare-python/compare/v5.5.0...v5.6.0)

### Breaking Changes

* **accounts:** The `create()` and `get()` methods have been removed from `client.accounts.subscriptions`. The `SubscriptionCreateParams` type is no longer available.
* **acm:** The `CertificateAuthority` type has been removed from `cloudflare.types.acm`.
* **certificate_authorities:** The `HostnameAssociation` type has been removed from `cloudflare.types.certificate_authorities`.
* **cloudforce_one:** The `indicator_types` sub-resource has been removed from `client.cloudforce_one.threat_events`. Use the new `client.cloudforce_one.threat_events.indicators.types` resource instead.
* **hyperdrive:** The pagination type for `client.hyperdrive.configs.list()` changed from `SyncSinglePage` to `SyncV4PagePaginationArray`. Code that relied on single-page iteration semantics may need adjustment.
* **load_balancers:** All methods on `client.load_balancers` now use a polymorphic `/{account_or_zone}/{account_or_zone_id}/load_balancers` endpoint pattern instead of the previous `/zones/{zone_id}/load_balancers`. Method signatures now accept either `account_id` or `zone_id`.
* **moq:** The `rotate()` method and `TokenRotateResponse` type have been removed from `client.moq.relays.tokens`. Use the new `create()`, `list()`, and `delete()` methods instead.
* **mtls_certificates:** The `MTLSCertificate` type has been replaced with `MTLSCertificateGetResponse`, `MTLSCertificateListResponse`, and `MTLSCertificateDeleteResponse`. The return types of `list()`, `delete()`, and `get()` have changed accordingly.
* **custom_certificates:** The `Status` type has been removed from `cloudflare.types.custom_certificates`.
* **ssl:** The `Host`, `Status`, and `ValidationMethod` types have been removed from `cloudflare.types.ssl`. The `hosts` parameter on `client.ssl.certificate_packs.create()` changed from `SequenceNotStr[Host]` to `SequenceNotStr[str]`.
* **zero_trust:** The `setup_flows` sub-resource has been removed from `client.zero_trust.casb.applications`. The `get()` method endpoint changed from `/accounts/{account_id}/one/applications/{slug}` to `/accounts/{account_id}/one/applications/{application_id}`.

### Features

* **analytics_query:** add analytics_query resource with `summary()`, `timeseries()`, and `top_n()` methods, plus `data_security.content_findings` and `data_security.findings` sub-resources ([7245164](https://github.com/cloudflare/cloudflare-python/commit/72451646f))
* **ai_audit:** add `client.ai_audit.robots` resource with `bulk_get()` and `get()` methods ([7245164](https://github.com/cloudflare/cloudflare-python/commit/72451646f))
* **cloudforce_one:** add extensive threat_events sub-resources: `aggregate`, `graphql`, `graph`, `queries`, `relationships`, `indicators` (with `aggregate`, `types`, `by_dataset`), `tags` (with `categories`, `indicators`), `target_industries`, `categories`, and `datasets` ([7245164](https://github.com/cloudflare/cloudflare-python/commit/72451646f))
* **magic_transit:** add `client.magic_transit.connectors.interrupts` resource with `create()` and `list()` methods ([7245164](https://github.com/cloudflare/cloudflare-python/commit/72451646f))
* **moq:** add `create()`, `list()`, and `delete()` methods to `client.moq.relays.tokens` ([7245164](https://github.com/cloudflare/cloudflare-python/commit/72451646f))
* **registrar_sandbox:** add registrar_sandbox resource with `check()`, `search()`, and sub-resources for `registrations`, `registration_status`, and `update_status` ([7245164](https://github.com/cloudflare/cloudflare-python/commit/72451646f))
* **zero_trust:** add `client.zero_trust.casb.applications.auth_methods` resource with `list()` method ([7245164](https://github.com/cloudflare/cloudflare-python/commit/72451646f))
* **zero_trust:** add comprehensive CASB posture management: `client.zero_trust.casb.posture` with sub-resources for `findings` (list, export, ignore, unignore, tune/reset severity), `finding_types`, `content`, `exports`, `remediations`, and `webhooks` ([7245164](https://github.com/cloudflare/cloudflare-python/commit/72451646f))

#### New Resources

**AIAudit Robots** (`client.ai_audit.robots`)

- `bulk_get(*, zone_id, **params) -> Optional[RobotBulkGetResponse]`
- `get(*, zone_id, **params) -> Optional[RobotGetResponse]`

New types:
```python
from cloudflare.types.ai_audit import RobotBulkGetResponse, RobotGetResponse
```

**MagicTransit Connector Interrupts** (`client.magic_transit.connectors.interrupts`)

- `create(connector_id, *, account_id, **params) -> InterruptCreateResponse`
- `list(connector_id, *, account_id) -> SyncSinglePage[InterruptListResponse]`

New types:
```python
from cloudflare.types.magic_transit.connectors import (
    InterruptCreateResponse,
    InterruptListResponse,
)
```

**CASB Application Auth Methods** (`client.zero_trust.casb.applications.auth_methods`)

- `list(application_id, *, account_id) -> AuthMethodListResponse`

New types:
```python
from cloudflare.types.zero_trust.casb.applications import AuthMethodListResponse
```

**MoQ Relay Tokens** (`client.moq.relays.tokens`)

- `create(relay_id, *, account_id, **params) -> Optional[TokenCreateResponse]`
- `list(relay_id, *, account_id) -> Optional[TokenListResponse]`
- `delete(jti, *, account_id, relay_id) -> TokenDeleteResponse`

New types:
```python
from cloudflare.types.moq.relays import (
    TokenCreateResponse,
    TokenListResponse,
    TokenDeleteResponse,
)
```

**CloudforceOne ThreatEvents Sub-Resources** (`client.cloudforce_one.threat_events`)

The following sub-resources were added under `client.cloudforce_one.threat_events`:

- `aggregate`
- `graphql`
- `graph`
- `queries`
- `relationships`
- `indicators` (with `aggregate`, `types`, `by_dataset` sub-resources)
- `tags` (with `categories`, `indicators` sub-resources)
- `target_industries`
- `categories`
- `datasets`

**AnalyticsQuery** (`client.analytics_query`)

Query analytics data across datasets with summary, timeseries, and top-N aggregations.

- `summary(dataset, *, account_id, **params) -> AnalyticsQuerySummaryResponse`
- `timeseries(dataset, *, account_id, **params) -> AnalyticsQueryTimeseriesResponse`
- `top_n(dataset, *, account_id, **params) -> SyncSinglePage[AnalyticsQueryTopNResponse]`

New types:
```python
from cloudflare.types import (
    AnalyticsQuerySummaryResponse,
    AnalyticsQueryTimeseriesResponse,
    AnalyticsQueryTopNResponse,
)
```

**DataSecurity ContentFindings** (`client.analytics_query.data_security.content_findings`)

- `top_n(*, account_id, **params) -> SyncSinglePage[ContentFindingTopNResponse]`

**DataSecurity Findings** (`client.analytics_query.data_security.findings`)

- `summary(*, account_id, **params) -> FindingSummaryResponse`
- `timeseries(*, account_id, **params) -> FindingTimeseriesResponse`

**RegistrarSandbox** (`client.registrar_sandbox`)

Sandbox environment for domain registration operations.

- `check(*, account_id, **params) -> RegistrarSandboxCheckResponse`
- `search(*, account_id, **params) -> RegistrarSandboxSearchResponse`

**Registrations** (`client.registrar_sandbox.registrations`)

- `create(*, account_id, **params) -> RegistrationCreateResponse`
- `list(*, account_id, **params) -> SyncCursorPagination[RegistrationListResponse]`
- `edit(domain_id, *, account_id, **params) -> RegistrationEditResponse`
- `get(domain_id, *, account_id) -> RegistrationGetResponse`

**RegistrationStatus** (`client.registrar_sandbox.registration_status`)

- `get(domain_name, *, account_id) -> RegistrationStatusGetResponse`

New types:
```python
from cloudflare.types.registrar_sandbox import RegistrationStatusGetResponse
```

**UpdateStatus** (`client.registrar_sandbox.update_status`)

- `get(domain_name, *, account_id) -> UpdateStatusGetResponse`

New types:
```python
from cloudflare.types.registrar_sandbox import UpdateStatusGetResponse
```

**CASB Posture** (`client.zero_trust.casb.posture`)

Full CASB posture management with findings, remediation, and webhook support.

- `posture.findings.list(*, account_id, **params) -> SyncV4PagePaginationArray[FindingListResponse]`
- `posture.findings.export(*, account_id, **params) -> Optional[FindingExportResponse]`
- `posture.findings.get(finding_id, *, account_id) -> Optional[FindingGetResponse]`
- `posture.findings.ignore(*, account_id, **params) -> Optional[FindingIgnoreResponse]`
- `posture.findings.unignore(*, account_id, **params) -> Optional[FindingUnignoreResponse]`
- `posture.findings.tune_severity(finding_id, *, account_id, **params) -> Optional[FindingTuneSeverityResponse]`
- `posture.findings.reset_severity(finding_id, *, account_id) -> Optional[FindingResetSeverityResponse]`
- `posture.findings.instances.list(*, account_id, **params) -> SyncV4PagePaginationArray[InstanceListResponse]`
- `posture.webhooks.create(*, account_id, **params) -> WebhookCreateResponse`
- `posture.webhooks.list(*, account_id) -> SyncSinglePage[WebhookListResponse]`
- `posture.remediations.jobs.create(*, account_id, **params) -> JobCreateResponse`
- `posture.remediations.jobs.list(*, account_id, **params) -> SyncV4PagePaginationArray[JobListResponse]`

New types:
```python
from cloudflare.types.zero_trust.casb.posture import (
    FindingListResponse,
    FindingExportResponse,
    FindingGetResponse,
    FindingIgnoreResponse,
    FindingUnignoreResponse,
    FindingTuneSeverityResponse,
    FindingResetSeverityResponse,
)
```

**FindingTypes** (`client.zero_trust.casb.posture.finding_types`)

- `list(*, account_id, **params) -> SyncV4PagePaginationArray[FindingTypeListResponse]`
- `get(finding_type_id, *, account_id) -> Optional[FindingTypeGetResponse]`

New types:
```python
from cloudflare.types.zero_trust.casb.posture import (
    FindingTypeListResponse,
    FindingTypeGetResponse,
)
```

**Content** (`client.zero_trust.casb.posture.content`)

- `list(*, account_id, **params) -> SyncV4PagePaginationArray[ContentListResponse]`
- `export(*, account_id, **params) -> Optional[ContentExportResponse]`

New types:
```python
from cloudflare.types.zero_trust.casb.posture import (
    ContentListResponse,
    ContentExportResponse,
)
```

**Exports** (`client.zero_trust.casb.posture.exports`)

- `list(*, account_id, **params) -> SyncV4PagePaginationArray[ExportListResponse]`
- `get(id, *, account_id) -> Optional[ExportGetResponse]`

New types:
```python
from cloudflare.types.zero_trust.casb.posture import (
    ExportListResponse,
    ExportGetResponse,
)
```

## 5.4.0 (2026-06-16)

Full Changelog: [v5.3.0...v5.4.0](https://github.com/cloudflare/cloudflare-python/compare/v5.3.0...v5.4.0)

### Features

* feat(ai_gateway): add custom_providers resource ([b982ff3](https://github.com/cloudflare/cloudflare-python/commit/b982ff391eab267ef16c719d0fc798d970292ed2))
* feat(api): map ipsec_tunnels/psk, sites/app_policies and cf1_sites for magic-on-ramps ([5d8e422](https://github.com/cloudflare/cloudflare-python/commit/5d8e422736d8e044d2c73065cbb8595573b2656d))
* feat(ct_alerter): add CT alerting subscription endpoint mappings ([5bb529c](https://github.com/cloudflare/cloudflare-python/commit/5bb529c2eabe4ce0c23cddbf8073c6a025dc3024))
* feat(tenants): add tenants resource SDK mapping [PT-2567] ([7677b4c](https://github.com/cloudflare/cloudflare-python/commit/7677b4c9566495b627b2cd90c9220268f9c6c46d))


### Chores

* **api:** update composite API spec ([2b0e738](https://github.com/cloudflare/cloudflare-python/commit/2b0e7388107c4110489484c2b38d4d9295b15af2))
* **api:** update composite API spec ([c7202c9](https://github.com/cloudflare/cloudflare-python/commit/c7202c9f9215a179b17362439c59b2e99566ab40))
* **api:** update composite API spec ([f94e4de](https://github.com/cloudflare/cloudflare-python/commit/f94e4de91421e736243f75eee987c888f06d8f0b))
* **api:** update composite API spec ([656f973](https://github.com/cloudflare/cloudflare-python/commit/656f97390d51da3e224e67f2305d6d46f7ae239a))
* **api:** update composite API spec ([05e802d](https://github.com/cloudflare/cloudflare-python/commit/05e802d1940e1b0bfe59a25c388b83f53edd238e))
* **api:** update composite API spec ([3817b57](https://github.com/cloudflare/cloudflare-python/commit/3817b571d02b2d23f087bbdb574853632d0bc1db))
* **api:** update composite API spec ([3e26c73](https://github.com/cloudflare/cloudflare-python/commit/3e26c739c29586ab33e05bb4dc6673d06748d013))
* **api:** update composite API spec ([0ff89f9](https://github.com/cloudflare/cloudflare-python/commit/0ff89f99ddfbaf078d3b20eada5fd2bc4139fc65))
* **api:** update composite API spec ([eeec39a](https://github.com/cloudflare/cloudflare-python/commit/eeec39a7ab478625ead7a1e6276c1c258ff574fd))
* **api:** update composite API spec ([45260b9](https://github.com/cloudflare/cloudflare-python/commit/45260b91f48ed30eb69f1ecd184f245b03a59146))
* **api:** update composite API spec ([72eb205](https://github.com/cloudflare/cloudflare-python/commit/72eb20551c18d2cae3dfbef960a12ff22b4a7de7))
* **internal:** codegen related update ([86bdf98](https://github.com/cloudflare/cloudflare-python/commit/86bdf981e37d4b028f4e555d7e0b1e76324f23b3))
