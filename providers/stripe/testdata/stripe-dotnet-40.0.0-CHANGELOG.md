## 40.0.0 - 2022-08-02

Major version release for API version 2022-08-01. Default API version changed to "2022-08-01".

Breaking changes that arose during code generation of the library that we postponed for the next major version. For changes to the SDK, read more detailed description at https://github.com/stripe/stripe-dotnet/wiki/Migration-guide-for-v40. For changes to the Stripe products, read more at https://stripe.com/docs/upgrades#2022-08-01.

"⚠️" symbol highlights breaking changes.

* [#2549](https://github.com/stripe/stripe-dotnet/pull/2549) API Updates
* [#2547](https://github.com/stripe/stripe-dotnet/pull/2547) Next major release changes

### Added
* Add `RefundAccountHolderName`, `RefundAccountHolderType`, and `RefundRoutingNumber` properties to `SourceAchCreditTransfer` entity.
* Add `DataString`, `NativeUrl`, and `StatementDescriptor` properties to `SourceAlipay` entity.
* Add `Description`, `Iin`, and `Issuer` properties to `SourceCard` and `SourceCardPresent` entities.
* Add `Reference` and `StatementDescriptor` properties to `SourceEps` entity.
* Add `StatementDescriptor` property to `SourceBancontact`,`SourceGiropay`, `SourceIdeal`, and `SourceSofort` entities.
* Add `RefundAccountHolderAddressCity`, `RefundAccountHolderAddressCountry`, `RefundAccountHolderAddressLine1`, `RefundAccountHolderAddressLine2`, `RefundAccountHolderAddressPostalCode`, `RefundAccountHolderAddressState`, `RefundAccountHolderName` and `RefundIban` properties to `SourceMultibanco` and `SourceSepaCreditTransfer` entities.
* Add `Name` property to `SourceThreeDSecure` entity.
* Add support for `ShippingCost` and `ShippingDetails` on `CheckoutSession`

### ⚠️ Changed
* Rename `BalanceDetails` to `BalanceIssuing`. Rename type of `Balance.Issuing` property.
* Change `Application` property in `BillingPortal.Configuration` entity to be expandable.
* Change `Delinquent` property in `Customer` entity to be nullable.
* Change `Amount` property in `Dispute` and `PaymentIntentAmountDetailsTip` entities to be non-nullable. The field is required.
* Change type of `Start` and `End` property in `InvoiceItemPeriod` and `InvoiceLineItemPeriod` entities to `DateTime`.
* Change `AmountSubtotal` and `AmountTotal` properties in `LineItem` entity to be non-nullable. The fields are required.
* Change `Director`, `Executive`, `Owner`, and `Representative` properties in `PersonRelationship` to be nullable.
* Change `Location` property in `Reader` and `Terminal.Reader` entities to be expandable.
* Change type of `ExpMonth` and `ExpYear` properties in `ExternalAccountUpdateOptions` and `TokenCardOptions` to `string`.
* Change type of `StartDate` property in `SubscriptionSchedulePhaseOptions` to union of date and string. This supports `"now"` as a valid value.
* Change type of `Timestamp` property in `UsageRecordCreateOptions` to union of date and string. This supports `"now"` as a valid value.

### ⚠️ Removed
* Remove `CustomerPaymentSettingsOptions`, `CustomerPaymentSettingsPaymentMethodOptionsBancontactOptions`, `CustomerPaymentSettingsPaymentMethodOptionsCardOptions`, and `CustomerPaymentSettingsPaymentMethodOptionsOptions`.
* Remove `IncorporationArticle`,  `IncorporationDocument`, `PaymentProviderTransfer`, and `ProductFeed` from `FilePurpose` constants. The values are no longer in use.
* Remove `DefaultSourceType` property from `Customer` entity. The field was undocumented and unsupported.
* Remove `AssociatedObjects` array property from `EphemeralKey` entity. The field was undocumented and unsupported.
* Remove `EphemeralKeyAssociatedObject` entity.
* Remove `Quantity` and `TrialEnd` properties in `CustomerCreateOptions`. The fields are not supported in the operation.
* Remove `TrialEnd` property in `CustomerUpdateOptions`. The property was not supported in the operation.
* Remove `Order` and `OrderItem` resources and APIs as those have been deprecated for multiple years.
  * Remove `OrderPaymentSucceeded`, `OrderUpdated`, and `OrderReturnCreated` event constants. These events are deprecated.
  * Remove `Order` and `OrderId` properties in `Charge` entity. The property was deprecated.
* Remove `QuoteFinalizeQuoteOptions`. The options weren't used anymore.
* Remove `RedirectUrl` from `LoginLinkCreateOptions`. The property was deprecated.

## 39.126.0 - 2022-07-26
* [#2544](https://github.com/stripe/stripe-dotnet/pull/2544) API Updates
  * Add support for `CustomerBalance` on `CheckoutSessionPaymentMethodOptionsOptions` and `CheckoutSessionPaymentMethodOptions`
