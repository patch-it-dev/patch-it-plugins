## 9.0.0 - 2022-08-02

Breaking changes that arose during code generation of the library that we postponed for the next major version. For changes to the SDK, read more detailed description at https://github.com/stripe/stripe-php/wiki/Migration-guide-for-v9. For changes to the Stripe products, read more at https://stripe.com/docs/upgrades#2022-08-01.

"⚠️" symbol highlights breaking changes.

* [#1344](https://github.com/stripe/stripe-php/pull/1344) API Updates
* [#1337](https://github.com/stripe/stripe-php/pull/1337) API Updates
* [#1273](https://github.com/stripe/stripe-php/pull/1273) Add some PHPDoc return types and fixes
* [#1341](https://github.com/stripe/stripe-php/pull/1341) Next major release changes

### Added
* Add `alternate_statement_descriptors`, `authorization_code`, and `level3` properties to `Charge` resource.
* Add `previewLines` method to `CreditNote` resource.
* Add `transfer_data` property to `Subscription` resource.
* Add `SOURCE_TYPE_FPX` constant to `Transfer` resource.
* Add new error code constants to `ErrorObject`.
* Add support for `shipping_cost` and `shipping_details` on `Checkout.Session`

### ⚠️ Changed
* Updated certificate bundle ([#1314](https://github.com/stripe/stripe-php/pull/1314))
* Add `params` parameter to `close` method in `Dispute` resource.

### ⚠️ Removed
* Remove deprecated `AlipayAccount`, `BitcoinReceiver`, `BitcoinTransaction`, `Recipient`, `RecipientTransfer`, and `ThreeDSecure` resources.
* Remove `CAPABILITY_CARD_PAYMENTS`, `CAPABILITY_LEGACY_PAYMENTS`, `CAPABILITY_PLATFORM_PAYMENTS`, `CAPABILITY_TRANSFERS`, `CAPABILITY_STATUS_ACTIVE`, `CAPABILITY_STATUS_INACTIVE`, and `CAPABILITY_STATUS_PENDING` constants from `Account` resource. Please use up-to-date values from https://stripe.com/docs/connect/account-capabilities.
* Remove `AssociatedObjects` array property from `EphemeralKey` resource. The field was undocumented and unsupported.
* Remove `details` method from `Card` resource. The endpoint was deprecated and no longer exists.
* Remove `recipient` property from `Card` resource. The property was deprecated.
* Remove ability to list `Card` resources for a particular `Recipient`.
* Remove `sources` property from `Card` resource. The property was deprecated.
* Remove `FAILURE_REASON` constant from `Refund` resource. The value was deprecated.
* Remove `Recipient` resource. The resource was deprecated.
* Remove `OrderItem` resource. The resource was deprecated.
* Remove `all` method from `LineItem`.
* Remove `cancel` method from `Transfer` and `TransferService`. This method is deprecated.
* Remove `allTransactions` method from `SourceService` service. Please use `allSourceTransactions` method instead.
* Remove `persons` method from `Account` resource. Please use `allPersons` method instead.
* Remove `sourceTransactions` method from `Source` resource. Please use `allSourceTransactions` method instead.
* Remove `usageRecordSummaries` method from `SubscriptionItem` resource. Please use `allUsageRecordSummaries` method instead.
* Remove `SOURCE_TYPE_ALIPAY_ACCOUNT` and `SOURCE_TYPE_FINANCING` constants from `Transfer` resource. The values were deprecated and are no longer in use.
* Remove deprecated error code constants from `ErrorObject`: `CODE_ACCOUNT_ALREADY_EXISTS`, `CODE_ORDER_CREATION_FAILED`, `CODE_ORDER_REQUIRED_SETTINGS`, `CODE_ORDER_STATUS_INVALID`, `CODE_ORDER_UPSTREAM_TIMEOUT`, and `CODE_UPSTREAM_ORDER_CREATION_FAILED`.
* Remove deprecated event constants from `Webhook`: `ISSUER_FRAUD_RECORD_CREATED`, ` ORDER_PAYMENT_FAILED`, `ORDER_PAYMENT_SUCCEEDED`, `ORDER_UPDATED`, `ORDER_RETURN_CREATED`, `PAYMENT_METHOD_CARD_AUTOMATICALLY_UPDATED`, `PING`, `PROMOTION_CODE_DELETED`, and `TREASURY_RECEIVED_CREDIT_REVERSED`. The events are deprecated and no longer sent by Stripe.

## 8.12.0 - 2022-07-25
* [#1332](https://github.com/stripe/stripe-php/pull/1332) API Updates
  * Add support for `default_currency` and `invoice_credit_balance` on `Customer`
