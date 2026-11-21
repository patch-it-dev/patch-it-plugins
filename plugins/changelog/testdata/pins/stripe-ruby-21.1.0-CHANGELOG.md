## <a id="21-1-0"></a>21.1.0 - 2026-11-18
This release changes the pinned API version to `2026-11-18.endive`.

* Fixes ListObject#next_page/#previous_page and SearchResultObject#next_search_result_page using the global requestor instead of the client that retrieved the page
* Preserve resource authorization scope across Marshal round-trips and event-related requests

  Resources retain their account and context when restored from a new `Marshal.dump`, including resources fetched from thin event notifications. Restoring an unscoped resource clears the loading process's account and context defaults. Full events bind to their payload's context when constructed, so related-object requests and subsequent mutations use that scope.

  Resources copy account and context selectors so mutating caller-provided strings cannot change their retained scope.

  Existing Marshal cache entries continue to load using the legacy behavior. Their original scope may already be missing, so clear cached SDK objects and re-fetch them after upgrading to benefit from this fix. Caching resource IDs together with their explicit context and reconstructing them through a scoped client is also supported.
* Keep API keys out of the inspect output of clients and event notifications

  `inspect` on `StripeClient`, `StripeConfiguration`, `APIRequestor`, event notification handlers, and `EventNotification` no longer includes API keys, webhook secrets, or proxy credentials, so passing these objects to `Logger`, `p`, or `pp` doesn't write credentials to logs.
* [#2026](https://github.com/stripe/stripe-ruby/pull/2026) Reuse connections across account and context scopes

  Account and context selectors no longer create separate connection pools when other configuration matches. Requests retain their own authorization headers, while proxy, TLS, certificate, and timeout differences still select separate pools.
* [#2026](https://github.com/stripe/stripe-ruby/pull/2026) Support nested request blocks and track responses across requestors

  Request blocks return the last response produced in that block, including requests through another client or a resource bound to a different requestor. Nested blocks track their own responses, including when they use the same requestor. Request routing continues to use each resource or client’s configured credentials and scope.
* Keep API keys out of YAML dumps of clients and event notifications

  `YAML.dump` of an `EventNotification`, `StripeClient`, or `APIRequestor` no longer includes API keys or proxy credentials. A notification restored from YAML keeps its fields but has no client, so `fetch_event` and `fetch_related_object` raise `Stripe::AuthenticationError`. Parse the original payload with `StripeClient#parse_event_notification` to make requests.
* [#2024](https://github.com/stripe/stripe-ruby/pull/2024) Update generated code
  * Add support for new resource `V2::Core::ClaimableSandbox`
  * Add support for `create`, `renew_onboarding_link`, and `retrieve` methods on resource `V2::Core::ClaimableSandbox`
  * Add support for `funding_sources` on `Charge::PaymentMethodDetail::Card::Wallet::Link` and `Charge::PaymentMethodDetail::Link`
  * Add support for `fingerprint` on `Charge::PaymentMethodDetail::Paypay`, `PaymentAttemptRecord::PaymentMethodDetail::Paypay`, and `PaymentRecord::PaymentMethodDetail::Paypay`
  * Add support for `address_collection_precision` on `Checkout::Session::AutomaticTax` and `Checkout::SessionCreateParams::AutomaticTax`
  * Change `Dispute::Evidence.appeal` to be required
  * Change `Issuing::Card::Shipping.delivered_at` to be required
  * Change `Issuing::Card::Shipping.shipped_at` to be required
  * Change `PaymentAttemptRecord::PaymentMethodDetail::Card.stored_credential_usage` and `PaymentRecord::PaymentMethodDetail::Card.stored_credential_usage` to be required
  * Add support for `car_rental_data` and `lodging_data` on `PaymentIntent::PaymentDetail`, `PaymentIntentCaptureParams::PaymentDetail`, `PaymentIntentConfirmParams::PaymentDetail`, `PaymentIntentCreateParams::PaymentDetail`, and `PaymentIntentUpdateParams::PaymentDetail`
  * Change type of `PaymentLinkCreateParams::PaymentIntentDatum.capture_method` from `enum('automatic'|'automatic_async'|'manual')` to `emptyable(enum('automatic'|'automatic_async'|'manual'))`
  * Add support for `capture_method` on `PaymentLinkUpdateParams::PaymentIntentDatum`
  * Add support for `loss_message` on `Product::MarketingFeature`, `ProductCreateParams::MarketingFeature`, and `ProductUpdateParams::MarketingFeature`
  * Add support for `application` on `Refund`
  * Add support for `bizum` and `upi` on `Refund::DestinationDetail`
  * Add support for `tax_invoice_month` on `Reporting::ReportRun::Parameter` and `Reporting::ReportRunCreateParams::Parameter`
  * Add support for `igic` on `Tax::Registration::CountryOption::E` and `Tax::RegistrationCreateParams::CountryOption::E`
  * Add support for `network_details` on `Treasury::InboundTransfer::FailureDetail`, `Treasury::InboundTransferFailParams::FailureDetail`, `Treasury::OutboundPayment::ReturnedDetail`, and `Treasury::OutboundTransfer::ReturnedDetail`
  * Add support for `failure_details` on `Treasury::InboundTransferReturnInboundTransferParams`
  * Add support for `account_dashboard` on `V2::Core::AccountLink::UseCase` and `V2::Core::AccountLinkCreateParams::UseCase`
  * Add support for `paypay_payments` on `V2::Core::Account::Configuration::Merchant::Capability`, `V2::Core::Account::Configuration::Merchant`, `V2::Core::AccountCreateParams::Configuration::Merchant::Capability`, `V2::Core::AccountCreateParams::Configuration::Merchant`, `V2::Core::AccountUpdateParams::Configuration::Merchant::Capability`, and `V2::Core::AccountUpdateParams::Configuration::Merchant`
  * Add support for `billie_payments`, `mb_way_payments`, `nz_bank_account_becs_debit_payments`, and `pix_payments` on `V2::Core::Account::Configuration::Merchant::Capability`, `V2::Core::AccountCreateParams::Configuration::Merchant::Capability`, and `V2::Core::AccountUpdateParams::Configuration::Merchant::Capability`
  * Add support for `additional_material` on `V2::Core::Account::Configuration::Merchant::KonbiniPayment`, `V2::Core::AccountCreateParams::Configuration::Merchant::KonbiniPayment`, and `V2::Core::AccountUpdateParams::Configuration::Merchant::KonbiniPayment`
  * Add support for event notifications `V2CoreClaimableSandboxClaimedEvent`, `V2CoreClaimableSandboxCreatedEvent`, `V2CoreClaimableSandboxExpiredEvent`, `V2CoreClaimableSandboxExpiringEvent`, and `V2CoreClaimableSandboxUpdatedEvent` with related object `V2::Core::ClaimableSandbox`

## <a id="21-0-0"></a>21.0.0 - 2026-11-11
* Improve webhook signature verification error messages
  - When diagnostic metadata is present in the `Stripe-Signature` header, errors help identify an incorrect signing secret or altered request body.
* [#2011](https://github.com/stripe/stripe-ruby/pull/2011) Accept binary webhook payloads with non-ASCII characters
  - Parse UTF-8 webhook payloads tagged as binary in `construct_event_without_verification` and `parse_event_notification_without_verification`.
* ⚠️ Omit API keys from Marshal and YAML dumps of StripeObjects

  Marshal and YAML dumps of StripeObjects no longer preserve API keys. Before making requests through a restored object, call `reattach_api_key(api_key)`. This also reattaches API keys to nested objects.

  See the [v21 migration guide](https://github.com/stripe/stripe-ruby/wiki/Migration-guide-for-v21#api-keys-are-no-longer-preserved-in-marshal-and-yaml-dumps-of-stripeobjects) for details.
* [#2022](https://github.com/stripe/stripe-ruby/pull/2022) Fix stripe_account and stripe_context scoping on Event#fetch_related_object

  When fetching a related object, directly-retrieved `Event` objects used the retaining client's own `stripe_context` rather than the event's `stripe_context`. `fetch_related_object` now routes through a requestor scoped to the event's context.  This mirrors how `EventNotification` scopes its client.
