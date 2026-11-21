## <a id="53-2-0"></a>53.2.0 - 2026-11-18
This release changes the pinned API version to `2026-11-18.endive`.

* Improve webhook signature verification error messages
  - When diagnostic metadata is present in the `Stripe-Signature` header, error messages help identify an incorrect signing secret or altered request body.
* Fix forward-compatible deserialization of unknown discriminated-union variants

  Preserve unknown string discriminators on concrete union bases with both Newtonsoft.Json and System.Text.Json, including safe serialization of fallback values.
* [#3491](https://github.com/stripe/stripe-dotnet/pull/3491) Prefix request_id (if available) in exception Message property
* [#3494](https://github.com/stripe/stripe-dotnet/pull/3494) Stop leaking a stale Stripe-Account when fetching an event's related object

  Adds `StripeClient.WithStripeContext()` to construct a `StripeClient` with the supplied `StripeContext` with `Stripe-Account` cleared.  Adds calls to `WithStripeClient` from `EventNotification.FromJson` and when processing a `V2.Core.Event` API response.
* Keep `StripeContext` segments when serializing it with Newtonsoft.Json or System.Text.Json

  `StripeContext` now serializes as its `Stripe-Context` string (e.g. `"acct_123/acct_456"`), matching the API's wire format, instead of `{"Segments": [...]}`, which didn't deserialize correctly. Payloads previously serialized as `{"Segments": [...]}` still deserialize correctly.
* [#3493](https://github.com/stripe/stripe-dotnet/pull/3493) Update generated code
  * Add support for new resource `V2.Core.ClaimableSandbox`
  * Add support for `Create`, `Get`, and `RenewOnboardingLink` methods on resource `V2.Core.ClaimableSandbox`
  * Add support for `FundingSources` on `Charge.PaymentMethodDetails.Card.Wallet.Link` and `Charge.PaymentMethodDetails.Link`
  * Add support for `Fingerprint` on `Charge.PaymentMethodDetails.Paypay`, `PaymentAttemptRecord.PaymentMethodDetails.Paypay`, and `PaymentRecord.PaymentMethodDetails.Paypay`
  * Add support for `AddressCollectionPrecision` on `Checkout.Session.AutomaticTax` and `CheckoutSessionAutomaticTaxOptions`
  * Add support for `CarRentalData` and `LodgingData` on `PaymentIntent.PaymentDetails` and `PaymentIntentPaymentDetailsOptions`
  * Change type of `PaymentLinkPaymentIntentDataOptions.CaptureMethod` from `enum('automatic'|'automatic_async'|'manual')` to `emptyable(enum('automatic'|'automatic_async'|'manual'))`
  * Add support for `CaptureMethod` on `PaymentLinkPaymentIntentDataOptions`
  * Add support for `LossMessage` on `Product.MarketingFeature`, `ProductMarketingFeatureOptions`, and `ProductMarketingFeaturesOptions`
  * Add support for `Application` on `Refund`
  * Add support for `Bizum` and `Upi` on `Refund.DestinationDetails`
  * Add support for `TaxInvoiceMonth` on `Reporting.ReportRun.Parameters` and `ReportingReportRunParametersOptions`
  * Add support for `Igic` on `Tax.Registration.CountryOptions.Es` and `TaxRegistrationCountryOptionsEsOptions`
  * Add support for `NetworkDetails` on `Treasury.InboundTransfer.FailureDetails`, `Treasury.OutboundPayment.ReturnedDetails`, `Treasury.OutboundTransfer.ReturnedDetails`, and `TreasuryInboundTransferFailureDetailsOptions`
  * Add support for `FailureDetails` on `Treasury.TestHelpersInboundTransferReturnInboundTransferOptions`
  * Add support for `AccountDashboard` on `V2.Core.AccountLink.UseCase` and `V2CoreAccountLinkUseCaseOptions`
  * Add support for `PaypayPayments` on `V2.Core.Account.Configuration.Merchant.Capabilities`, `V2.Core.Account.Configuration.Merchant`, `V2CoreAccountConfigurationMerchantCapabilitiesOptions`, and `V2CoreAccountConfigurationMerchantOptions`
  * Add support for `BilliePayments`, `MbWayPayments`, `NzBankAccountBecsDebitPayments`, and `PixPayments` on `V2.Core.Account.Configuration.Merchant.Capabilities` and `V2CoreAccountConfigurationMerchantCapabilitiesOptions`
  * Add support for `AdditionalMaterial` on `V2.Core.Account.Configuration.Merchant.KonbiniPayments` and `V2CoreAccountConfigurationMerchantKonbiniPaymentsOptions`
  * Add support for event notifications `V2CoreClaimableSandboxClaimedEvent`, `V2CoreClaimableSandboxCreatedEvent`, `V2CoreClaimableSandboxExpiredEvent`, `V2CoreClaimableSandboxExpiringEvent`, and `V2CoreClaimableSandboxUpdatedEvent` with related object `V2.Core.ClaimableSandbox`

## <a id="52-4-2"></a>52.4.2 - 2026-09-09
* [#3445](https://github.com/stripe/stripe-dotnet/pull/3445) Validate that webhook secrets are non-empty

## <a id="52-0-0"></a>52.0.0 - 2026-06-05
This release **doesn't** change the pinned API version; it still uses `2026-05-27.dahlia`.

We're doing an out-of-band-major to update a field type that changed. If you're not using `tax_details`, this is a no-op release when compared with the last one. If you _are_ using `tax_details` its type has changed slightly and you'll have to update your code when upgrading.

* [#3394](https://github.com/stripe/stripe-dotnet/pull/3394) Add "source" field to user-agent header
* ⚠️ [#3396](https://github.com/stripe/stripe-dotnet/pull/3396) Make `tax_rate.tax_details` expandable
