## <a id="87-2-0"></a>87.2.0 - 2026-11-18
This release changes the pinned API version to `2026-11-18.endive`.

* Improve webhook signature verification error messages
  - When diagnostic metadata is present in the `Stripe-Signature` header, error messages help identify an incorrect signing secret or altered request body.
* Upgrade decimal dependency to v1.5.0
  - Decimal values with excessively large positive or negative exponents now return decoding errors.
* Honor each client's `EnableTelemetry` setting
  - Separate the `X-Stripe-Client-User-Agent` cache so that clients with different telemetry settings coexist in the same process.
* Send an `Idempotency-Key` on write requests made without params
  - `RawRequest` and the deprecated resource packages (e.g. `account.New(nil)`) now generate an `Idempotency-Key` for `POST`, `PUT`, `PATCH`, and `DELETE` requests when params is `nil`, and reuse it on retries. Previously these requests were sent and retried without a key.
* [#2496](https://github.com/stripe/stripe-go/pull/2496) Update generated code
  * Add support for new resource `V2CoreClaimableSandbox`
  * Add support for `Get`, `New`, and `RenewOnboardingLink` methods on resource `V2CoreClaimableSandbox`
  * Add support for `FundingSources` on `ChargePaymentMethodDetailsCardWalletLink` and `ChargePaymentMethodDetailsLink`
  * Add support for new value `tron` on enums `ChargePaymentMethodDetailsCrypto.Network`, `PaymentAttemptRecordPaymentMethodDetailsCrypto.Network`, and `PaymentRecordPaymentMethodDetailsCrypto.Network`
  * Add support for `Fingerprint` on `ChargePaymentMethodDetailsPaypay`, `PaymentAttemptRecordPaymentMethodDetailsPaypay`, and `PaymentRecordPaymentMethodDetailsPaypay`
  * Add support for `AddressCollectionPrecision` on `CheckoutSessionAutomaticTaxParams` and `CheckoutSessionAutomaticTax`
  * Add support for `CarRentalData` and `LodgingData` on `PaymentIntentCapturePaymentDetailsParams`, `PaymentIntentConfirmPaymentDetailsParams`, `PaymentIntentPaymentDetailsParams`, and `PaymentIntentPaymentDetails`
  * Change type of `PaymentLinkPaymentIntentDataParams.CaptureMethod` from `enum('automatic'|'automatic_async'|'manual')` to `emptyable(enum('automatic'|'automatic_async'|'manual'))`
  * Add support for `CaptureMethod` on `PaymentLinkPaymentIntentDataParams`
  * Add support for `LossMessage` on `ProductMarketingFeatureParams`, `ProductMarketingFeature`, and `ProductMarketingFeaturesParams`
  * Add support for `Application` on `Refund`
  * Add support for `Bizum` and `Upi` on `RefundDestinationDetails`
  * Add support for `TaxInvoiceMonth` on `ReportingReportRunParametersParams` and `ReportingReportRunParameters`
  * Add support for `Igic` on `TaxRegistrationCountryOptionsEsParams` and `TaxRegistrationCountryOptionsEs`
  * Add support for new value `igic` on enum `TaxRegistrationCountryOptionsEs.Type`
  * Add support for new values `stripe_s700_devkit`, `stripe_s710_devkit`, `stripe_t600_devkit`, `verifone_ux700_devkit`, and `verifone_v660p_devkit` on enum `TerminalReader.DeviceType`
  * Add support for `NetworkDetails` on `TestHelpersTreasuryInboundTransferFailFailureDetailsParams`, `TreasuryInboundTransferFailureDetails`, `TreasuryOutboundPaymentReturnedDetails`, and `TreasuryOutboundTransferReturnedDetails`
  * Add support for `FailureDetails` on `TestHelpersTreasuryInboundTransferReturnInboundTransferParams`
  * Add support for `AccountDashboard` on `V2CoreAccountLinkUseCaseParams` and `V2CoreAccountLinkUseCase`
  * Add support for new value `account_dashboard` on enum `V2CoreAccountLinkUseCase.Type`
  * Add support for `PaypayPayments` on `V2CoreAccountConfigurationMerchantCapabilitiesParams`, `V2CoreAccountConfigurationMerchantCapabilities`, `V2CoreAccountConfigurationMerchantParams`, and `V2CoreAccountConfigurationMerchant`
  * Add support for `BilliePayments`, `MbWayPayments`, `NzBankAccountBECSDebitPayments`, and `PixPayments` on `V2CoreAccountConfigurationMerchantCapabilitiesParams` and `V2CoreAccountConfigurationMerchantCapabilities`
  * Add support for `AdditionalMaterial` on `V2CoreAccountConfigurationMerchantKonbiniPaymentsParams` and `V2CoreAccountConfigurationMerchantKonbiniPayments`
  * Add support for new values `billie_payments`, `mb_way_payments`, `nz_bank_account_becs_debit_payments`, `paypay_payments`, and `pix_payments` on enums `V2CoreAccountFutureRequirementsEntryImpactRestrictsCapability.Capability` and `V2CoreAccountRequirementsEntryImpactRestrictsCapability.Capability`
  * Add support for new values `billie_payments`, `mb_way_payments`, `nz_bank_account_becs_debit_payments`, `paypay_payments`, and `pix_payments` on enum `EventsV2CoreAccountIncludingConfigurationMerchantCapabilityStatusUpdatedEvent.UpdatedCapability`
  * Add support for event notifications `V2CoreClaimableSandboxClaimedEvent`, `V2CoreClaimableSandboxCreatedEvent`, `V2CoreClaimableSandboxExpiredEvent`, `V2CoreClaimableSandboxExpiringEvent`, and `V2CoreClaimableSandboxUpdatedEvent` with related object `V2CoreClaimableSandbox`
  * Add support for error code `inactive_api_operation_confirmation_required` on `Error`, `InvoiceLastFinalizationError`, `PaymentIntentLastPaymentError`, `SetupAttemptSetupError`, `SetupIntentLastSetupError`, `StripeError`, and `TerminalReaderActionApiError`
  * Add support for error codes `claimable_sandbox_creation_rate_limit_exceeded` and `claimable_sandbox_testing_creation_rate_limit_exceeded` on `RateLimitError`

## <a id="86-4-2"></a>86.4.2 - 2026-09-09
* [#2429](https://github.com/stripe/stripe-go/pull/2429) Validate that webhook secrets are non-empty

## <a id="86-0-0"></a>86.0.0 - 2026-06-05
This release **doesn't** change the pinned API version; it still uses `2026-05-27.dahlia`.

We're doing an out-of-band-major to update a field type that changed. If you're not using `tax_details`, this is a no-op release when compared with the last one. If you _are_ using `tax_details` its type has changed slightly and you'll have to update your code when upgrading.

* [#2367](https://github.com/stripe/stripe-go/pull/2367) Remove `Limit`, `StartingAfter`, and `EndingBefore` fields for `List` methods that do not accept those fields
  - Fixes a bug where `Limit`, `StartingAfter`, and `EndingBefore` were embedded in `CapabilityListParams` and `ReportingReportTypeListParams` even though they are not valid parameters and would have always resulted in a 400 from the Stripe API if set. If you were including them before in either of those 2 structs, you can safely remove them.
* [#2370](https://github.com/stripe/stripe-go/pull/2370) Add "source" field to user-agent header
* ⚠️ [#2375](https://github.com/stripe/stripe-go/pull/2375) Make `tax_rate.tax_details` expandable
