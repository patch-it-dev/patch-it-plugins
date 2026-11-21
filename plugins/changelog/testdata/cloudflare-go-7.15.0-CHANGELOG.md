## 7.15.0 (2026-11-06)

Full Changelog: [v7.14.0...v7.15.0](https://github.com/cloudflare/cloudflare-go/compare/v7.14.0...v7.15.0)

### Breaking Changes

See the [v7.15.0 Migration Guide](./docs/migration-guides/v7.15.0-migration-guide.md) for before/after code examples
and actions needed for each change.

* **accounts:** the `Type` field is removed from `AccountUpdateParams`, along with the `AccountUpdateParamsType`
  enum (`standard`, `enterprise`). `Accounts.Update()` no longer accepts a `type`.
* **ssl:** `CertificatePacks.New()` response is simplified. `CertificatePackNewResponse` drops its `Certificates`,
  `DCVDelegationRecords`, `ValidationErrors`, and `ValidationRecords` fields, and the nested
  `CertificatePackNewResponseCertificate`, `CertificatePackNewResponseCertificatesGeoRestrictions`,
  `CertificatePackNewResponseCertificatesGeoRestrictionsLabel`, `CertificatePackNewResponseDCVDelegationRecord`,
  `CertificatePackNewResponseValidationError`, and `CertificatePackNewResponseValidationRecord` types are removed.
* **rules:** the `ListCursor` and `ListCursorParam` types are removed.
* **tenants:** `Accounts.List()` return type changed from `*pagination.V4PagePaginationArray[AccountListResponse]`
  to `*pagination.SinglePage[AccountListResponse]`. The endpoint no longer uses page-based pagination.

### Features

* **workers:** add `Observability.Agents.Traces` sub-resource with `List()`, `Events()`, and `Get()`
  (`/accounts/{account_id}/workers/observability/agents/traces`)
* **workers:** add `Observability.Issues` sub-resource with `Update()`, `List()`, `Add()`, `Get()`, `Group()`,
  `Notify()`, `Occurrences()`, `Remove()`, `Summary()`, and `Ungroup()`, plus the nested `Issues.Automations`
  sub-resource with `New()`, `Update()`, `List()`, `Delete()`, and `Get()`
  (`/accounts/{account_id}/workers/observability/issues`)
* **zero_trust:** add `Organizations.TrustSettings` sub-resource with `Update()`, `Delete()`, `Get()`, and `Replace()`
  (`/accounts/{account_id}/access/organizations/trust_settings`)
* **zones:** add `OhttpGateway` sub-resource with `New()`, `Delete()`, and `Get()`
  (`/zones/{zone_id}/settings/ohttp-gateway`)
* **realtime_kit:** add `Livestreams.GetAppByID()`, `UpdateAppByID()`, `ReplaceParticipant()`,
  `RevokeParticipantToken()`, `GetAllWebhooks()`, `MuteAllParticipants()`, `MuteParticipants()`,
  `GetLivestreamSessionDetailsForMeeting()`, and `GetLivestreamSessionsForSession()` methods

## 7.13.1 (2026-10-20)

Full Changelog: [v7.13.0...v7.13.1](https://github.com/cloudflare/cloudflare-go/compare/v7.13.0...v7.13.1)

Users are encouraged to build with the latest Go patch release. Rebuild your application with Go >= 1.26.10 or >= 1.27.3.
This library sets no TLS trust policy of its own — trust is resolved by `crypto/x509` in the toolchain that
compiles your binary, so the minimum-version change above does not itself provide this fix. Note that Go
caches the system certificate pool per process, so a restart is required after updating any trust store.

### Changes

* The minimum supported Go version is raised from 1.22 to 1.26. Go 1.25 and earlier are no longer supported.
  Projects whose own `go.mod` declares a lower version will need to raise it to build against this release.
