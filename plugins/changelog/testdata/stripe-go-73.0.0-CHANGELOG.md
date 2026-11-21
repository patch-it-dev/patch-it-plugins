## 73.0.0 - 2022-08-02
Major version release for API version 2022-08-01. Default API version changed to "2022-08-01".

Breaking changes that arose during code generation of the library that we postponed for the next major version. For changes to the SDK, read more detailed description at https://github.com/stripe/stripe-go/wiki/Migration-guide-for-v73. For changes to the Stripe products, read more at https://stripe.com/docs/upgrades#2022-08-01.

"⚠️" symbol highlights breaking changes.

* [#1513](https://github.com/stripe/stripe-go/pull/1513) API Updates
* [#1512](https://github.com/stripe/stripe-go/pull/1512) Next major release changes

### Added

- Add `CheckoutSessionSetupIntentDataParams.Metadata`.
- Add Invoice `UpcomingLines` method.
- Add `ShippingCost` and `ShippingDetails` properties to `CheckoutSession` resource.
- Add `CheckoutSessionShippingCostTax` and `CheckoutSessionShippingCost` classes
- Add `IssuingCardCancellationReasonDesignRejected` constant to `IssuingCardCancellationReason`.
- Add `Validate` field to `Customer` resource.
- Add `Validate` field to `PaymentSourceParams`.
- Add `SetupAttemptPaymentMethodDetailsCardThreeDSecureResultExempted` constant in `SetupAttemptPaymentMethodDetailsCardThreeDSecureResult`.
- Add `SKUPackageDimensionsParams` and `SKUPackageDimensions`.
- Add dedicated structs for different payment sources and transfers.
- Add `Subscription.DeleteDiscount` methods.
- Add `SubscriptionItemUsageRecordSummariesParams`
- Add `UsageRecordSummary` `UsageRecordSummaries`, and `UsageRecordSummaryList` methods in `SubscriptionItem`
- Add `SubscriptionSchedulePhaseBillingCycleAnchor`, `SubscriptionSchedulePhaseBillingCycleAnchorAutomatic`, and `SubscriptionSchedulePhaseBillingCycleAnchorPhaseStart`
- Add `SubscriptionSchedulePhaseInvoiceSettings` and `SubscriptionSchedulePhaseInvoiceSettingsParams `
- `TerminalLocation` `UnmarshalJSON` - make `TerminalLocation` expandable
* Add support for new value `invalid_tos_acceptance` on enums `AccountFutureRequirementsErrorsCode`, `AccountRequirementsErrorsCode`, `CapabilityFutureRequirementsErrorsCode`, `CapabilityRequirementsErrorsCode`, `PersonFutureRequirementsErrorsCode`, and `PersonRequirementsErrorsCode`
* Add support for `ShippingCost` and `ShippingDetails` on `CheckoutSession`

### ⚠️ Changed

- Rename files to be consistent with the library's naming conventions.
    - `fee.go` to `applicationfee.go` 
    - `fee/client.go` to `applicationfee/client.go` 
    - `sub.go` to `subscription.go` 
    - `sub/client.go` to `subscription/client.go` 
    - `subitem.go` to `subscriptionitem.go` 
    - `subitem/client.go` to `subscriptionitem/client.go` 
    - `subschedule.go` to `subscriptionschedule.go` 
    - `subschedule/client.go` to `subscriptionschedule/client.go` 
    - `reversal.go` to `transferreversal.go` 
    - `reversal/client.go` to `transferreversal/client.go` 

- Change resource names on `client#API` to be plural to be consistent with the library's naming conventions: 
- Rename structs, fields, enums, and methods to be consistent with the library's naming conventions and with the other Stripe SDKs.
  - `Ach` to `ACH`
  - `Acss` to `ACSS`
  - `Bic` to `BIC`
  - `Eps` to `EPS`
  - `FEDEX` to `FedEx`
  - `Iban` to `IBAN`
  - `Ideal` to `IDEAL`
  - `Sepa` to `SEPA`
  - `Wechat` to `WeChat`
  - `ExternalAccount` to `AccountExternalAccount`
  - `InvoiceLine` to `InvoiceLineItem`
  - `Person` structs/enums to use `Person` prefix
  - and others (see Migration guide)

- Change types of various fields in `Account`, `ApplicationFee`, `BalanceTransaction`, `BillingPortalConfiguration`, `Card`, `Charge`, `Customer`, `Discount`, `Invoice`, `Issuing Card`,  `Issuing Dispute `, `Mandate `, `PaymentIntent`, `PaymentMethod`, `Payout`, `Plan `, `Plan `, `Refund`, `SetupIntent`, `Source`, `Source`, `Subscription`, `SubscriptionItem`, `SubscriptionSchedule`, `Terminal ConnectionToken`, `Terminal Location`, `Terminal Reader `, `Topup`, and `Transfer` (see Migration guide).

- Update the Webhook `ConstructEvent,` `ConstructEventIgnoringTolerance` and `ConstructEventWithTolerance` functions to return an error when the webhook event's API version does not match the stripe-go library API version.
- Update `ErrorType`and `ErrorCode` values.
- Move `BalanceTransaction` iterator from `balance.go` to `balancetransaction.go`
- Fix `BalanceTransactionSource` `UnmarshalJSON` for when `BalanceTransactionSource.Type == "transfer_reversal"` (previously, we were checking if `Type == "reversal"`, which was always false)
- For BankAccount and Card client methods, check that exactly one of `params.Account` and `params.Customer` is set (previously they could both be set, but only one would be used, and it was different between BankAccount and Card)
- Replace `CardVerification` with field-specific enums (with the same values)
- Move `Del` from `discount/client.go` to `customer/client.go` and rename to `DeleteDiscount`
- Move `DelSub` from `discount/client.go` to `subscription/client.go` and rename to `DeleteDiscount`
- Add separate parameter struct for CreditNote `ListPreviewLines` (renamed to `PreviewLines`) method (`[CreditNoteLineItemListPreviewParams -> CreditNotePreviewParams].Lines` `CreditNoteLineParams` -> `CreditNotePreviewLineParams`)
- Replace `FeeRefundParams.ApplicationFee` with `FeeRefundParams.Fee` and `FeeRefundParams.ID`
- Add separate parameter struct for Invoice `GetNext` (renamed to `Upcoming`) method (`InvoiceUpcomingParams`, and nested params `InvoiceUpcomingLinesInvoiceItemPriceDataParams`, `InvoiceUpcomingLinesInvoiceItemDiscountParams`, `InvoiceUpcomingLinesDiscountParams`, `InvoiceUpcomingLinesInvoiceItemPeriodParams`). `Upcoming`-only fields `Coupon`, `CustomerDetails`, `InvoiceItems`, `Subscription`, `SubscriptionBillingCycleAnchor`, `Schedule`, `SubscriptionBillingCycleAnchor`, `SubscriptionBillingCycleAnchorNow`, `SubscriptionBillingCycleAnchorUnchanged`, `SubscriptionCancelAt`, `SubscriptionCancelAtPeriodEnd`, `SubscriptionCancelNow`, `SubscriptionDefaultTaxRates`, `SubscriptionItems`, `SubscriptionProrationBehavior`, `SubscriptionProrationDate`, `SubscriptionStartDate`, `SubscriptionTrialEnd`, `SubscriptionTrialEndNow`, and `SubscriptionTrialFromPlan` are removed from `InvoiceParams`.
- Add separate structs for `BillingDetails` and `BillingDetailsParams`: `PaymentMethodBillingDetails`, `PaymentMethodBillingDetailsParams`
- Add separate structs for `PaymentMethodCardNetwork`: `PaymentMethodCardNetworksAvailable`, `PaymentMethodCardNetworksPreferred`

### Deprecated

- The `SKU` resource has been deprecated. This will be replaced by https://stripe.com/docs/api/orders_v2.

### ⚠️ Removed

- Remove the legacy Orders API
- Remove `AccountCapability` enum definition. This was not referenced in the library.
- Remove `UnmarshalJSON` for resources that are not expandable: `BillingPortalSession`, `Capability`, `CheckoutSession`, `FileLink`, `InvoiceItem`, `LineItem`, `Person`, `WebhookEndpoint`
- Remove `AccountRejectReason` (was only referenced in `account/client_test.go`, actual `AccountRejectParams.Reason` is `*string`)
- Remove `AccountParams.RequestedCapabilities` (use Capabilities instead: https://stripe.com/docs/connect/account-capabilities)
- Remove `AccountSettingsParams.Dashboard` and `AccountSettingsDashboardParams` (Note: `Dashboard` are still available on `AccountSettings`, but it's not available as parameters for any of the methods)
- Remove `AccountCompany.RegistrationNumber` (Note: `RegistrationNumber` is still available on `AccountCompanyParams`, but is not returned in the response)
- Remove `BalanceTransactionStatus`. It was meant to be an enum, but none of the enum values were defined, so it was just an alias for string.
- Remove `CardParams.AccountType`. `AccountType` does not exist on any client method for Card. It does on BankAccount, which is similar.
- Remove `id` param from CheckoutSessions `ListLineItems`. Use `CheckoutSessionListLineItemsParams.Session` instead.
- Remove `CheckoutSessionLineItemPriceDataRecurringParams.AggregateUsage`, `CheckoutSessionLineItemPriceDataRecurringParams.TrialPeriodDays`, and `CheckoutSessionLineItemPriceDataRecurringParams.UsageType`
- Remove `CheckoutSessionPaymentIntentDataParams.Params`, `CheckoutSessionSetupIntentDataParams.Params`, `CheckoutSessionSubscriptionDataParams.Params`. `Params` should only be embedded in root method struct, and has extraneous fields not applicable to child/sub structs.
- Remove `CheckoutSessionTotalDetailsBreakdownTax.TaxRate`. Use `CheckoutSessionTotalDetailsBreakdownTax.Rate`
- Remove `CheckoutSessionTotalDetailsBreakdownTax.Deleted`
- Remove `CustomerParams.Token`
- Remove `Discount` `APIResource` embed
- Remove `DiscountParams`
- Remove `FilePurposeFoundersStockDocument` (`"founders_stock_document"` option for `File.Purpose`)
- Remove `InvoiceParams.Paid`. Use `invoice.status` to check for status. `invoice.status` is a read-only field.
- Remove `InvoiceParams.SubscriptionPlan` and `InvoiceParams.SubscriptionQuantity` (note: these would have been on `InvoiceUpcomingParams`)
- Remove `InvoiceListLinesParams.Customer` and `InvoiceListLinesParams.Subscription` (these are not available for Invoice `ListLines`, but are available for `List`)
- Remove `IssuingAuthorizationRequestHistoryViolatedAuthorizationControlEntity` and `IssuingAuthorizationRequestHistoryViolatedAuthorizationControlName` (unused enums)
- Remove `IssuingCardSpendingControlsParams.SpendingLimitsCurrency`. `issuing_card` has `currency`, and `issuing_card.spending_controls.spending_limits.amount` will use that currency
- Remove `IssuingDisputeEvidenceServiceNotAsDescribed.ProductDescription`, `IssuingDisputeEvidenceServiceNotAsDescribed.ProductType`, `IssuingDisputeEvidenceServiceNotAsDescribedParams.ProductDescription`, `IssuingDisputeEvidenceServiceNotAsDescribedParams.ProductType`, and `IssuingDisputeEvidenceServiceNotAsDescribedProductType`. `issuing_dispute.evidence.service_not_as_described` does not have `product_description` or `product_type`. `issuing_dispute.evidence.canceled` does.
- Remove `LineItemTax.TaxRate`. Use `LineItemTax.Rate` instead.
- Remove `LineItem.Deleted`
- Remove `LoginLink.RedirectURL`
- Remove `PaymentIntentOffSession` (unused enum)
- Remove `PaymentIntentConfirmParams.PaymentMethodTypes`
- Remove `PaymentMethodFPX.TransactionID`
- Remove `Payout.BankAccount` and `Payout.Card` (These fields were never populated, use `PayoutDestination.BankAccount` and `PayoutDestination.Card` instead)
- Remove `PlanParams.ProductID`. Use `PlanParams.Product.ID` instead.
- Remove `Shipping` and `ShippingRate` properties from `CheckoutSession` resource. Please use `ShippingCost` and `ShippingDetails` properties instead.
- Remove `DefaultCurrency` property from `Customer` resource. Please use `Currency` property instead.
- Remove `Updated` and `UpdatedBy` from `RadarValueList`
- Remove `Name` from `RadarValueListItem`
- Remove `ReviewReasonType` type from `Review` resource. Use `ReviewReason` instead
- Remove `SetupIntentCancellationReasonFailedInvoice` and `SetupIntentCancellationReasonFraudulent` values from `SetupIntentCancellationReason`
- Remove `SigmaScheduledQueryRun.Query`. The field was invalid
- Remove `SKUParams.Description` and `SKU.Description`
- Remove `SourceMandateAcceptanceStatus`, `SourceMandateAcceptanceStatusAccepted`, `SourceMandateAcceptanceStatusRefused`, `SourceMandateNotificationMethod`, `SourceMandateNotificationMethodEmail`, `SourceMandateNotificationMethodManual`, and `SourceMandateNotificationMethodNone`
- Remove `Source.TypeData` and SourceParams and replace with payment method-specific fields (AUBECSDebit, Bancontact, Card, CardPresent, EPS, Giropay, IDEAL, Klarna, Multibanco, P24, SEPACreditTransfer, SEPADebit, Sofort, ThreeDSecure, Wechat) and `Source.AppendTo` method
- Remove `SourceTransaction.CustomerData`. The field was deprecated
- Remove `SourceTransaction.TypeData` and `SourceTransaction.UnmarshalJSON`. Use payment specific fields - Remove `ACHCreditTransfer`, `CHFCreditTransfer`, `GBPCreditTransfer`, `PaperCheck`, and `SEPACreditTransfer`
- Remove `SubscriptionPaymentBehavior`, `SubscriptionPaymentBehaviorAllowIncomplete`, `SubscriptionPaymentBehaviorErrorIfIncomplete`, and `SubscriptionPaymentBehaviorPendingIfIncomplete`
- Remove `SubscriptionProrationBehavior`, `SubscriptionProrationBehaviorAlwaysInvoice`, `SubscriptionProrationBehaviorCreateProrations`, and `SubscriptionProrationBehaviorNone`
- Remove `SubscriptionStatusAll`
- Remove `SubscriptionParams.Card`, `SubscriptionParams.Plan`, and `SubscriptionParams.Quantity`
- Remove `Subscription.Plan` and `Subscription.Quantity`
- Remove `SubscriptionItemParams.ID`. The field was deprecated
- Remove `SubscriptionSchedulePhaseAddInvoiceItemPriceDataRecurringParams` and `SubscriptionSchedulePhaseAddInvoiceItemPriceDataParams`
- Remove `Del` method on `TaxRate`
- Remove `TerminalReaderGetParams`. Use `TerminalReaderParams` instead.
- Remove `TerminalReaderList.Location` and `TerminalReaderList.Status` (Not available for the list, but is available for individual `TerminalReader`s in `TerminalReaderList.Data`)
- Remove `Token.Email` and `TokenParams.Email`
- Remove `TopupParams.SetSource`
- Remove `WebhookEndpointListParams.Created` and `WebhookEndpointListParams.CreatedRange` (use `StartingAfter` from `ListParams`)
- Remove `WebhookEndpoint.Connected`

## 72.122.0 - 2022-07-26
* [#1508](https://github.com/stripe/stripe-go/pull/1508) API Updates
  * Add support for new value `exempted` on enums `ChargePaymentMethodDetailsCardThreeDSecureResult` and `SetupAttemptPaymentMethodDetailsCardThreeDSecureResult`
  * Add support for `CustomerBalance` on `CheckoutSessionPaymentMethodOptionsParams` and `CheckoutSessionPaymentMethodOptions`
