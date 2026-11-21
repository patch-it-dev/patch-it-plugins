## 5.29.0 (2026-11-10)

Full Changelog: [v5.28.1...v5.29.0](https://github.com/cloudflare/terraform-provider-cloudflare/compare/v5.28.1...v5.29.0)

### ⚠ BREAKING CHANGES

- **zone_dns_settings, account_dns_settings:** the deprecated `foundation_dns` attribute has been removed. On `cloudflare_zone_dns_settings` a configuration that still sets it now fails with an "Unsupported argument" error; remove it and use `nameservers.type` (`cloudflare.advanced` to enable Advanced Nameservers, `cloudflare.standard` to disable). On `cloudflare_account_dns_settings` the field lived inside the nested `zone_defaults` object and is dropped silently on upgrade. No state upgrader is required.
- **zero_trust_gateway_policy:** the `rule_settings.audit_ssh` block has been removed; the Audit SSH action is no longer offered. Remove `audit_ssh` from any policy configuration. An automatic state migration drops a previously stored `audit_ssh` value.
- **worker_version:** the Durable Object binding `retry` block (`retry.max_attempts`, `retry.timeout_ms`) has been removed from the `cloudflare_worker_version` resource and `cloudflare_worker_versions` data source, following its removal from the API. Remove it from any configuration.
- **zero_trust_access_application, zero_trust_access_policy:** the MFA authenticator value `ssh_piv_key` has been replaced. Use `piv_key` for PIV smart cards and the new `ssh_fido2_key` for FIDO2 SSH authenticators.
- **precursor:** `default_mode` has been renamed to `mode` and `enforcement_rules` to `rules`, and the enforcement enum values `min-friction` and `max-security` are replaced by `off`, `monitor`, and `arm`. The import ID is now `<zone_id>/<cloudflare_version>` instead of `<zone_id>`. Rename the arguments, update any enum values and expressions, and adjust `terraform import` commands.
- **ai_search_instance:** the import ID is now `<account_id>/<namespace>/<id>` instead of `<account_id>/<id>`. Update any `terraform import` commands to include the namespace.
- **organization:** the `profile` and `parent` attributes now force replacement — changing either destroys and recreates the organization. Review plans before applying.
- **share_recipient:** the `organization_id` attribute has been removed; configurations that still set it now fail with an "Unsupported argument" error. Remove it.

### Notes

- **cloudflare-go:** the Go SDK dependency is updated to v7.15.0.
- **zero_trust_access_policy:** the computed `app_count` attribute has been removed from the resource because it changes outside Terraform and produced perpetual diffs. The attribute remains available on the `cloudflare_zero_trust_access_policy` data source and list data source; update any expressions that referenced `cloudflare_zero_trust_access_policy.<name>.app_count`.

### Features

#### New Resources

- **cloudflare_ai_security:** manage AI Security for Apps on a zone.
- **cloudflare_ai_security_custom_topics:** manage custom topic categories for AI Security content detection on a zone.
- **cloudflare_log_explorer_dataset:** manage a Log Explorer dataset for an account or zone, with optional field selection and filtering. Supports `terraform import`.
- **cloudflare_zero_trust_casb_finding_type:** manage a custom CASB finding type, including its asset class, vendor, and Rego detection expression. Supports `terraform import`.
- **cloudflare_zero_trust_dlp_proximity_validator:** manage a DLP proximity validator that raises match confidence using a keyword list. Supports `terraform import`.
- **cloudflare_zero_trust_organization_trust_settings:** manage cross-account Access trust for an organization, granting trust to all accounts or a specified set. Supports `terraform import`.
- **cloudflare_zero_trust_cws_basic_auth_credential, cloudflare_zero_trust_cws_basic_auth_secret, cloudflare_zero_trust_cws_deployment, cloudflare_zero_trust_cws_domain_acl, cloudflare_zero_trust_cws_policy, cloudflare_zero_trust_cws_policy_set, cloudflare_zero_trust_cws_rego_metadata, cloudflare_zero_trust_cws_rego_module, cloudflare_zero_trust_cws_tenant_structure, cloudflare_zero_trust_cws_tenant_structure_node, cloudflare_zero_trust_cws_workload:** manage CWS deployments, policy sets and Rego policies, domain ACLs, basic-auth credentials and secrets, tenant structures and nodes, and workloads. All support `terraform import`.

#### New Data Sources

- Singular and list data sources for each new resource above (`cloudflare_ai_security`, `cloudflare_ai_security_custom_topics`, `cloudflare_zero_trust_casb_finding_type`, the `cloudflare_zero_trust_cws_*` set, `cloudflare_zero_trust_dlp_proximity_validator`, `cloudflare_zero_trust_organization_trust_settings`); `cloudflare_log_explorer_dataset` ships a list data source.
- **cloudflare_zero_trust_gateway_dns_destination_ips:** new list data source returning an account's Gateway DNS destination IP pairs (`primary_ip`, `backup_ip`, `pair_type`).

#### New Attributes

- **ruleset:** `webmcp_enabled` and `webmcp_packs` on configuration rules.
- **worker_version:** `container` added to the `exports` schema.

#### New Accepted Values

- **zero_trust_access_application:** `session_duration` is now supported for infrastructure applications; it sets the maximum connection duration and must use whole seconds expressed with `s`, `m`, or `h`.
- **zero_trust_access_policy:** `connection_rules.rdp` clipboard format attributes accept `file` in addition to `text`.
- **account_dns_settings:** `zone_defaults.nameservers.type` accepts `cloudflare.advanced`.

### Bug Fixes

- **account_subscription, stream_download:** drop model fields missing from the schema that could break state reads.
- **account_token:** send null when `expires_on`, `not_before`, or `condition` is removed.
- **ai_gateway_dynamic_routing:** preserve response envelopes and normalize version-graph responses.
- **ai_search_instance:** restore the generated resource and data sources; keep the data source `account_id` optional.
- **certificate_pack:** warn when deletion times out; detect soft deletion.
- **custom_ssl:** rotate `legacy_custom` certificates via PATCH instead of DELETE+POST.
- **dns_record:** require `content` at plan time for record types that use it (A, AAAA, CNAME, MX, NS, OPENPGPKEY, PTR, TXT); ignore a trailing dot when comparing CNAME content.
- **email_routing_settings:** apply `support_subaddress` on create and update.
- **list, list_item:** reject unnormalized IPs and CIDRs at plan time; check item attribute conflicts faster.
- **load_balancer:** preserve safe defaults in plans; give the v4 override `session_affinity_attributes` its own source model.
- **logpush_job:** skip destination config refresh.
- **managed_transforms:** remove duplicate schema fields and use the generated SDK response union.
- **observatory_scheduled_test:** support an optional region during import.
- **snippet:** do not crash when reading a snippet after import.
- **workers_script:** plan `startup_time_ms` as unknown when a deploy is planned; preserve `bindings_inherit` during the v0 state upgrade.
- **zone:** correctly upgrade v5-shaped state stored at schema version 0, guarding against missing raw state, so early v5 state (v5.0–v5.16) is not re-transformed.
- **zone_cache_variants:** send the full value on update.
- **zone_dns_settings:** remove `foundation_dns` to match the DNS settings API.
- **zero_trust_access_application:** stop inline policy selectors leaking across rules on refresh; nullify all-null `landing_page_design` on refresh; default and normalize the login prompt on import and against false/null drift.
- **zero_trust_access_group, zero_trust_access_policy:** stop selectors leaking across rules on refresh.
- **zero_trust_access_identity_provider:** default `login_page_auto_prompt` on read.
- **zero_trust_device_custom_profile:** match split-tunnel entries by address or host.
- **zero_trust_device_default_profile:** keep `policy_id` from state in plans.
- **zero_trust_gateway_policy:** preserve precedence during state migration.
- **zero_trust_organization:** preserve omitted security settings.
- **zero_trust_tunnel_cloudflared_config:** stop `origin_request {}` to null drift.

## 5.28.1 (2026-10-20)

Full Changelog: [v5.28.0...v5.28.1](https://github.com/cloudflare/terraform-provider-cloudflare/compare/v5.28.0...v5.28.1)

### Changes

- the provider is now built with Go 1.27.3, which distrusts a set of intermediate certificate authorities that have been revoked for CA compromise. The distrust list is compiled into the provider binary, so affected chains are rejected regardless of the host's system trust store. Upgrade to this release to pick up the change; no configuration changes are required.
- **cloudflare-go**: the Go SDK dependency is updated to v7.13.1.
- **cloudflare-go**: the unused v6 Go SDK dependency is removed. The `api_shield_discovery_operation` list data source now uses v7.
