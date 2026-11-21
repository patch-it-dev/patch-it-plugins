## <a id="23-1-0"></a>23.1.0 - 2026-11-18
This release changes the pinned API version to `2026-11-18.endive`.

* Preserve custom `api_base` configuration across paginated requests

  Ensure `nextPage()`, `previousPage()`, and `autoPagingIterator()` continue using the `api_base` configured on `StripeClient`.
* Keep API keys out of serialized and dumped clients and event notifications

  `serialize()`, `print_r()`, and `var_dump()` of an `EventNotification` or `StripeClient` no longer include the API key. `print_r()` and `var_dump()` show the client's key redacted. An unserialized notification keeps its fields but has no client, so `fetchEvent()` and `fetchRelatedObject()` throw `\Stripe\Exception\AuthenticationException`. Parse the original payload with `StripeClient::parseEventNotification()` to make requests. An unserialized client has no API key, including clients serialized by earlier SDK versions.
* [#2204](https://github.com/stripe/stripe-php/pull/2204) Update generated code
  * Add support for new resource `V2.Core.ClaimableSandbox`
  * Add support for `create`, `renew_onboarding_link`, and `retrieve` methods on resource `V2.Core.ClaimableSandbox`
  * Add support for `funding_sources` on `Charge.payment_method_details.card.wallet.link` and `Charge.payment_method_details.link`
  * Add support for new value `tron` on enums `Charge.payment_method_details.crypto.network`, `PaymentAttemptRecord.payment_method_details.crypto.network`, and `PaymentRecord.payment_method_details.crypto.network`
  * Add support for `fingerprint` on `Charge.payment_method_details.paypay`, `PaymentAttemptRecord.payment_method_details.paypay`, and `PaymentRecord.payment_method_details.paypay`
  * Add support for `address_collection_precision` on `Checkout.Session.automatic_tax` and `Checkout\Session.create().$params.automatic_tax`
  * Change `Dispute.evidence.appeal` to be required
  * Change `Issuing.Card.shipping.delivered_at` to be required
  * Change `Issuing.Card.shipping.shipped_at` to be required
  * Change `PaymentAttemptRecord.payment_method_details.card.stored_credential_usage` and `PaymentRecord.payment_method_details.card.stored_credential_usage` to be required
  * Add support for `car_rental_data` and `lodging_data` on `PaymentIntent.capture().$params.payment_detail`, `PaymentIntent.confirm().$params.payment_detail`, `PaymentIntent.create().$params.payment_detail`, `PaymentIntent.payment_details`, and `PaymentIntent.update().$params.payment_detail`
  * Change type of `PaymentLink.create().$params.payment_intent_datum.capture_method` from `enum('automatic'|'automatic_async'|'manual')` to `emptyable(enum('automatic'|'automatic_async'|'manual'))`
  * Add support for `capture_method` on `PaymentLink.update().$params.payment_intent_datum`
  * Add support for `loss_message` on `Product.create().$params.marketing_feature`, `Product.marketing_features[]`, and `Product.update().$params.marketing_feature`
  * Add support for `application` on `Refund`
  * Add support for `bizum` and `upi` on `Refund.destination_details`
  * Add support for `tax_invoice_month` on `Reporting.ReportRun.parameters` and `Reporting\ReportRun.create().$params.parameter`
  * Add support for `igic` on `Tax.Registration.country_options.es` and `Tax\Registration.create().$params.country_option.e`
  * Add support for new value `igic` on enum `Tax.Registration.country_options.es.type`
  * Add support for new values `stripe_s700_devkit`, `stripe_s710_devkit`, `stripe_t600_devkit`, `verifone_ux700_devkit`, and `verifone_v660p_devkit` on enum `Terminal.Reader.device_type`
  * Add support for `network_details` on `Treasury.InboundTransfer.failure_details`, `Treasury.OutboundPayment.returned_details`, `Treasury.OutboundTransfer.returned_details`, and `Treasury\InboundTransfer.fail().$params.failure_detail`
  * Add support for `failure_details` on `Treasury\InboundTransfer.return_inbound_transfer().$params`
  * Add support for `account_dashboard` on `V2.Core.AccountLink.use_case` and `V2\Core\AccountLink.create().$params.use_case`
  * Add support for new value `account_dashboard` on enum `V2.Core.AccountLink.use_case.type`
  * Add support for `paypay_payments` on `V2.Core.Account.configuration.merchant.capabilities`, `V2.Core.Account.configuration.merchant`, `V2\Core\Account.create().$params.configuration.merchant.capability`, `V2\Core\Account.create().$params.configuration.merchant`, `V2\Core\Account.update().$params.configuration.merchant.capability`, and `V2\Core\Account.update().$params.configuration.merchant`
  * Add support for `billie_payments`, `mb_way_payments`, `nz_bank_account_becs_debit_payments`, and `pix_payments` on `V2.Core.Account.configuration.merchant.capabilities`, `V2\Core\Account.create().$params.configuration.merchant.capability`, and `V2\Core\Account.update().$params.configuration.merchant.capability`
  * Add support for `additional_material` on `V2.Core.Account.configuration.merchant.konbini_payments`, `V2\Core\Account.create().$params.configuration.merchant.konbini_payment`, and `V2\Core\Account.update().$params.configuration.merchant.konbini_payment`
  * Add support for new values `billie_payments`, `mb_way_payments`, `nz_bank_account_becs_debit_payments`, `paypay_payments`, and `pix_payments` on enums `V2.Core.Account.future_requirements.entries[].impact.restricts_capabilities[].capability` and `V2.Core.Account.requirements.entries[].impact.restricts_capabilities[].capability`
  * Add support for new values `billie_payments`, `mb_way_payments`, `nz_bank_account_becs_debit_payments`, `paypay_payments`, and `pix_payments` on enum `EventsV2CoreAccountIncludingConfigurationMerchantCapabilityStatusUpdatedEvent.updated_capability`
  * Add support for event notifications `V2CoreClaimableSandboxClaimedEvent`, `V2CoreClaimableSandboxCreatedEvent`, `V2CoreClaimableSandboxExpiredEvent`, `V2CoreClaimableSandboxExpiringEvent`, and `V2CoreClaimableSandboxUpdatedEvent` with related object `V2.Core.ClaimableSandbox`
  * Add support for error code `inactive_api_operation_confirmation_required` on `Invoice.last_finalization_error`, `PaymentIntent.last_payment_error`, `SetupAttempt.setup_error`, `SetupIntent.last_setup_error`, `StripeError`, and `Terminal.Reader.action.api_error`

## <a id="23-0-0"></a>23.0.0 - 2026-11-11
* Improve webhook signature verification error messages
  - When diagnostic metadata is present in the `Stripe-Signature` header, error messages help identify an incorrect signing secret or altered request body.
* ⚠️ Omit API keys from native PHP serialization

  Native PHP serialization of SDK resources no longer preserves API keys. Deserialization also discards keys embedded by earlier SDK versions. Before making requests through a restored resource, call `reattachApiKey($apiKey)`. This also reattaches API keys to nested objects.

  See the [v23 migration guide](https://github.com/stripe/stripe-php/wiki/Migration-guide-for-v23#api-keys-no-longer-survive-native-php-serialization) for details.
* [#2187](https://github.com/stripe/stripe-php/pull/2187) Fix issue where v2 search requests can send multiple API requests
* Preserve event context when fetching related objects from a directly retrieved Event
* ⚠️ StripeClient no longer inherits the legacy global Stripe::$accountId

  ⚠️ `StripeClient` no longer inherits the global account set by `Stripe::setAccountId()`. Previously, a client without a configured `stripe_account` could send requests under an unrelated account set elsewhere in the process.

  If you relied on this behavior, pass `['stripe_account' => $accountId]` when constructing the client or making a request.

  An explicit `['stripe_account' => null]` now clears the account for a request, overriding both the client's configured account and the global account. Legacy static API calls still inherit the global account when `stripe_account` is omitted.

## <a id="21-0-0"></a>21.0.0 - 2026-07-15
This release **does not** change the pinned API version. It's still `2026-06-24.dahlia`. 

We're releasing it as a major out of an abundance of caution, but it should be functionally a patch release for most users. See below.

* [#2095](https://github.com/stripe/stripe-php/pull/2095) Remove unused Retry-After header support
* ⚠️ [#2097](https://github.com/stripe/stripe-php/pull/2097) Correctly type properties on `ErrorObject`
  * the properties of `ErrorObject` were typed as `string` when many of them should have been `null|string`. If you (or your typechecker) were treating these as plain strings, you'll need to be more defensive in your code.
  * to be clear: no runtime code has changed, we've just made the types more accurate. We didn't want to break any builds in a patch version, so this is released as a major
* [#2098](https://github.com/stripe/stripe-php/pull/2098) Replace source hash with Telemetry UUID
