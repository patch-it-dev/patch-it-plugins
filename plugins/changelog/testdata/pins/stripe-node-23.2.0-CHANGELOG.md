## <a id="23-2-0"></a>23.2.0 - 2026-11-18
This release changes the pinned API version to `2026-11-18.endive`.

* Improve webhook signature verification error messages
  - When diagnostic metadata is present in the `Stripe-Signature` header, error messages help identify an incorrect signing secret or altered request body.
* Accept RFC 3339 strings for V2 date-time range filters

  V2 date-time range filters now accept RFC 3339 strings. Numeric bounds remain in place to avoid breaking the interface, although the API does not accept them.
* [#2924](https://github.com/stripe/stripe-node/pull/2924) Fix Stripe-Context and Stripe-Account for event notification fetch helpers
  - Updates `fetchEvent` to call `withStripeContext` before issuing API call to fetch the event.
  - Updates `addFetchRelatedObjectIfNeeded` (used by `events.retrieve()`/`list()`) the same way.
* [#2927](https://github.com/stripe/stripe-node/pull/2927) Prefix requestId to error message field if a requestId exists
* Fix multipart encoding of `undefined`, `null`, and `Date` params in `files.create`
  - Multipart requests now follow the same rules as other requests: `undefined` params are omitted, `null` is sent as an empty string, and `Date` values are sent as Unix timestamps. Previously `undefined` and `null` were sent as the strings `"undefined"` and `"null"`, and `Date` params were dropped.
* [#2919](https://github.com/stripe/stripe-node/pull/2919) Update generated code
  * Add support for new resource `V2.Core.ClaimableSandbox`
  * Add support for `create`, `renew_onboarding_link`, and `retrieve` methods on resource `V2.Core.ClaimableSandbox`
  * Add support for `funding_sources` on `Charge.payment_method_details.card.wallet.link` and `Charge.payment_method_details.link`
  * Add support for new value `tron` on enums `Charge.payment_method_details.crypto.network`, `PaymentAttemptRecord.payment_method_details.crypto.network`, and `PaymentRecord.payment_method_details.crypto.network`
  * Add support for `fingerprint` on `Charge.payment_method_details.paypay`, `PaymentAttemptRecord.payment_method_details.paypay`, and `PaymentRecord.payment_method_details.paypay`
  * Add support for `address_collection_precision` on `Checkout.Session.automatic_tax` and `Checkout.SessionCreateParams.automatic_tax`
  * Change `Dispute.evidence.appeal` to be required
  * Change `Issuing.Card.shipping.delivered_at` to be required
  * Change `Issuing.Card.shipping.shipped_at` to be required
  * Change `PaymentAttemptRecord.payment_method_details.card.stored_credential_usage` and `PaymentRecord.payment_method_details.card.stored_credential_usage` to be required
  * Add support for `car_rental_data` and `lodging_data` on `PaymentIntent.payment_details`, `PaymentIntentCaptureParams.payment_details`, `PaymentIntentConfirmParams.payment_details`, `PaymentIntentCreateParams.payment_details`, and `PaymentIntentUpdateParams.payment_details`
  * Change type of `PaymentLinkCreateParams.payment_intent_data.capture_method` from `enum('automatic'|'automatic_async'|'manual')` to `emptyable(enum('automatic'|'automatic_async'|'manual'))`
  * Add support for `capture_method` on `PaymentLinkUpdateParams.payment_intent_data`
  * Add support for `loss_message` on `Product.marketing_features[]`, `ProductCreateParams.marketing_features[]`, and `ProductUpdateParams.marketing_features[]`
  * Add support for `application` on `Refund`
  * Add support for `bizum` and `upi` on `Refund.destination_details`
  * Add support for `tax_invoice_month` on `Reporting.ReportRun.parameters` and `Reporting.ReportRunCreateParams.parameters`
  * Add support for `igic` on `Tax.Registration.country_options.es` and `Tax.RegistrationCreateParams.country_options.es`
  * Add support for new value `igic` on enums `Tax.Registration.country_options.es.type` and `Tax.RegistrationCreateParams.country_options.es.type`
  * Add support for new values `stripe_s700_devkit`, `stripe_s710_devkit`, `stripe_t600_devkit`, `verifone_ux700_devkit`, and `verifone_v660p_devkit` on enums `Terminal.Reader.device_type` and `Terminal.ReaderListParams.device_type`
  * Add support for `network_details` on `Treasury.InboundTransfer.failure_details`, `Treasury.InboundTransferFailParams.testHelpers.failure_details`, `Treasury.OutboundPayment.returned_details`, and `Treasury.OutboundTransfer.returned_details`
  * Add support for `failure_details` on `Treasury.InboundTransferReturnInboundTransferParams.testHelpers`
  * Add support for new value `2026-11-18.endive` on enum `WebhookEndpointCreateParams.api_version`
  * Add support for `account_dashboard` on `V2.Core.AccountLink.use_case` and `V2.Core.AccountLinkCreateParams.use_case`
  * Add support for new value `account_dashboard` on enums `V2.Core.AccountLink.use_case.type` and `V2.Core.AccountLinkCreateParams.use_case.type`
  * Add support for `paypay_payments` on `V2.Core.Account.configuration.merchant.capabilities`, `V2.Core.Account.configuration.merchant`, `V2.Core.AccountCreateParams.configuration.merchant.capabilities`, `V2.Core.AccountCreateParams.configuration.merchant`, `V2.Core.AccountUpdateParams.configuration.merchant.capabilities`, and `V2.Core.AccountUpdateParams.configuration.merchant`
  * Add support for `billie_payments`, `mb_way_payments`, `nz_bank_account_becs_debit_payments`, and `pix_payments` on `V2.Core.Account.configuration.merchant.capabilities`, `V2.Core.AccountCreateParams.configuration.merchant.capabilities`, and `V2.Core.AccountUpdateParams.configuration.merchant.capabilities`
  * Add support for `additional_material` on `V2.Core.Account.configuration.merchant.konbini_payments`, `V2.Core.AccountCreateParams.configuration.merchant.konbini_payments`, and `V2.Core.AccountUpdateParams.configuration.merchant.konbini_payments`
  * Add support for new values `billie_payments`, `mb_way_payments`, `nz_bank_account_becs_debit_payments`, `paypay_payments`, and `pix_payments` on enums `V2.Core.Account.future_requirements.entries[].impact.restricts_capabilities[].capability` and `V2.Core.Account.requirements.entries[].impact.restricts_capabilities[].capability`
  * Add support for new values `billie_payments`, `mb_way_payments`, `nz_bank_account_becs_debit_payments`, `paypay_payments`, and `pix_payments` on enum `EventsV2CoreAccountIncludingConfigurationMerchantCapabilityStatusUpdatedEvent.updated_capability`
  * Add support for event notifications `V2CoreClaimableSandboxClaimedEvent`, `V2CoreClaimableSandboxCreatedEvent`, `V2CoreClaimableSandboxExpiredEvent`, `V2CoreClaimableSandboxExpiringEvent`, and `V2CoreClaimableSandboxUpdatedEvent` with related object `V2.Core.ClaimableSandbox`
  * Add support for error code `inactive_api_operation_confirmation_required` on `Invoice.last_finalization_error`, `PaymentIntent.last_payment_error`, `SetupAttempt.setup_error`, `SetupIntent.last_setup_error`, `StripeError`, and `Terminal.Reader.action.api_error`
  * Add support for error codes `claimable_sandbox_creation_rate_limit_exceeded` and `claimable_sandbox_testing_creation_rate_limit_exceeded` on `RateLimitError`

## <a id="23-0-1"></a>23.0.1 - 2026-10-22
* [#2885](https://github.com/stripe/stripe-node/pull/2885) Reject ASCII-whitespace-only webhook secrets
  - Reject webhook signing secrets made entirely of ASCII whitespace instead of using them as HMAC keys.
* [#2884](https://github.com/stripe/stripe-node/pull/2884) Add API v2 Search support

  Adds `V2SearchResult<T>` and support for auto pagination of search results.
* [#2895](https://github.com/stripe/stripe-node/pull/2895) Fix notification handling and discarded fetch retries
  - Fix `EventNotificationHandler.handle()` to verify signatures asynchronously, so it works with async crypto providers (e.g. the web platform).
  - Fall back to the full `EventNotification` union type for `on()` callbacks when a specific notification type can't be narrowed, instead of resolving to `never`.
  - Dispose discarded `fetch` responses before retrying a request, aborting the fetch and closing the unread body.
* Fix lastResponse.statusCode for fetch responses
  - Populate `lastResponse.statusCode` with the HTTP status for successful JSON responses when using the fetch HTTP client.

## <a id="22-0-0"></a>22.0.0 - 2026-04-02
This release uses the same pinned API version to `2026-03-25.dahlia` as the last major release. The breaking changes in this release are prefixed with ⚠️ below. There's also a [detailed migration guide](https://github.com/stripe/stripe-node/wiki/Migration-guide-for-v22) to simplify your upgrade process.

* ⚠️ [#2619](https://github.com/stripe/stripe-node/pull/2619) Improved TypeScript support in the Node SDK
  * Moved the types from the partially manually maintained and partially generated types folder to be inline with the implementation in TypeScript files.
    * Removed top-level “stripe” ambient module. This allows import aliasing for the stripe package.
    * ⚠️ `Stripe.StripeContext` is no longer exported as a type. Use `Stripe.StripeContextType` instead.
    * ⚠️ `Stripe.errors.StripeError` is no longer a type. Use `typeof Stripe.errors.StripeError` or `Stripe.ErrorType` instead.
    * ⚠️ CJS entry point no longer exports .default or .Stripe as separate properties.
    * ⚠️ Stripe import is now a true ES6 class. Use `new Stripe()` to create a StripeClient instead of calling it:
  ```javascript
  // Before
  const stripeClient = Stripe("sk_test_...");

  // After
  const stripeClient = new Stripe("sk_test_...");
  ```
* [#2619](https://github.com/stripe/stripe-node/pull/2619) Improved TypeScript support in the Node SDK
* [#2635](https://github.com/stripe/stripe-node/pull/2635) Updated stripe.spec.ts test and constructEvent.tolerance type
* [#2638](https://github.com/stripe/stripe-node/pull/2638) Converted V2/Amount.ts to V2/V2Amount.ts
* ⚠️ [#2643](https://github.com/stripe/stripe-node/pull/2643) Removed per-request host override. To use a custom host, set it in the client configuration. All requests from that client will use that host.

  Before:
  ```ts
  import Stripe from 'stripe';
  const stripe = new Stripe('sk_test_...');

  const customer = await stripe.customers.create({
    email: 'customer@example.com',
  }, {host: 'example.com'});
  ```

  After:
  ```ts
  import Stripe from 'stripe';
  const stripe = new Stripe('sk_test_...', {host: 'example.com'});

  // goes to example.com
  const customer = await stripe.customers.create({
    email: 'customer@example.com',
  });
  ```
* ⚠️ [#2645](https://github.com/stripe/stripe-node/pull/2645) Remove `stripeMethod` and standardize how function args are handled (including removing callback support)
  - ⚠️ Refactor how incoming method arguments are parsed. Type signatures for API methods should be _much_ more accurate and reliable now
    - ⚠️ Remove support for providing callbacks to API methods. Use `async / await` instead
    - ⚠️ Remove support for passing a plain API key as a function arg. If supplied on a per-request basis, it should be in the `RequestOptions` under the `apiKey` property
    - ⚠️ Keys from `params` and `options` objects are no longer mixed. If present on a method, `RequestParams` must always come first and `RequestOptions` must always come second. To supply options without params, pass `undefined` as the first argument explicitly
    - ⚠️ Removed methods from `StripeResource`: `createFullPath`, `createResourcePathWithSymbols`, `extend`, `method` and `_joinUrlParts`. These were mostly intended for internal use and we no longer need them

  As a result, the following call patterns are no longer supported:

  ```ts
  stripe.customers.retrieve('cus_123', 'sk_test_123')
  stripe.customers.create({name: 'david', host: 'example.com'}, 'sk_test_123')
  stripe.customers.create({apiKey: 'sk_test_123'})
  stripe.customers.list(customers => {
    // do something with customers
  })
  ```

  If those look familiar, head over to the [migration guide](https://github.com/stripe/stripe-node/wiki/Migration-guide-for-v22) to update your code.
