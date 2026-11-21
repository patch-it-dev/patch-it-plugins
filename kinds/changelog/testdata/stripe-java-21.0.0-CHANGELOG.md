## 21.0.0 - 2022-08-02

Major version release for API version 2022-08-01. Default API version changed to "2022-08-01".

Breaking changes that arose during code generation of the library that we postponed for the next major version. For changes to the SDK, read more detailed description at https://github.com/stripe/stripe-java/wiki/Migration-guide-for-v21. For changes to the Stripe products, read more at https://stripe.com/docs/upgrades#2022-08-01.

"⚠️" symbol highlights breaking changes.

* [#1409](https://github.com/stripe/stripe-java/pull/1409) API Updates
* [#1407](https://github.com/stripe/stripe-java/pull/1407) Next major release changes

### Added
* Add `ApiKeyMissingException`.
* Add `validate` field to `CustomerCreateParams`.
* Add `validate` field to `CustomerUpdateParams`.
* Add `validate` field to `PaymentSourceCollectionCreateParams`.
* Add `shippingCost`, `shippingDetails`, and `shippingOptions` properties to `checkout.Session` resource.
* Add support for `shipping_cost` and `shipping_details` on `Checkout.Session`
* Add support for new value `2022-08-01` on enum `WebhookEndpointCreateParams.api_version`

### ⚠️ Removed

- Removed deprecated `AlipayAccount` and related classes.
- Removed deprecated `BitcoinReceiver` and related classes.
- Removed deprecated `BitcoinTransaction` and related classes.
- Removed deprecated `Recipient` and related classes.
- Removed deprecated `IssuerFraudRecord` and related classes.
- Removed deprecated `Order` and related classes.
- Removed deprecated `ThreeDSecure` and related classes.
- Removed unused `QuoteFinalizeParams`, `QuoteCollectionListParams` classes.
- Removed unused `BillingDetails` class.
- Removed `CashBalance.retrieveCashBalance` method. Prefer `CashBalance.retrieve`.
- Removed `Rule.getDeleted` method. The property was never populated and always had the default value of `false`.
- Removed `LineItem.getDeleted` method. The property was never populated and always had the default value of `false`.
- Removed public constructors from `TestHelper` inner classes and made them static.
- Removed instance `Account.refresh`, `Account.disconnect` methods.
- Removed deprecated `TREASURY__RECEIVED_CREDIT__REVERSED`, `TREASURY__RECEIVED_DEBIT__CREATED`, `ORDER__UPDATED`, `ORDER_RETURN__CREATED`, `ORDER__PAYMENT_SUCCEEDED`, `TRANSFER__FAILED`, `TRANSFER__PAID` webhook events.
- Removed deprecated `LoginLink.redirectUrl` property.
- Removed deprecated `Charge.order` property.
- Removed deprecated `Card.recipient` property.
- Removed `defaultCurrency` property from `Customer` resource. Please use `Currency` property instead.
- Removed `shipping` and `shippingOptions` properties from `checkout.Session` resource. Please use `shippingCost`, `shippingDetails`, and `shippingOptions` properties instead.
- Removed `InitiatingPaymentMethodDetails` class

### ⚠️ Changed

- Default API version changed to "2022-07-28".
- Changed `CashBalance.retrieve` method from an instance to a static.
- Renamed `PaymentIntent.FinancialAddresses` class to `FinancialAddress`.
- Check that apiKey is set (either globally or via request options) in `StripeCollection.autoPagingIterable` and `StripeSearchResult.autoPagingIterable`, and throw an exception if it is not.
- Changed type of `businessType` field in `AccountCreateParams` from `Object` to `BusinessType`


## 20.136.0 - 2022-07-26
* [#1406](https://github.com/stripe/stripe-java/pull/1406) API Updates
  * Add support for `customer_balance` on `Checkout.Session.payment_method_options` and `CheckoutSessionCreateParams.payment_method_options`
  * Add support for new value `customer_balance` on enum `CheckoutSessionCreateParams.payment_method_types[]`
  * Add support for new values `en-CA` and `fr-CA` on enums `OrderCreateParams.payment.settings.payment_method_options.klarna.preferred_locale`, `OrderUpdateParams.payment.settings.payment_method_options.klarna.preferred_locale`, `PaymentIntentConfirmParams.payment_method_options.klarna.preferred_locale`, `PaymentIntentCreateParams.payment_method_options.klarna.preferred_locale`, and `PaymentIntentUpdateParams.payment_method_options.klarna.preferred_locale`
* [#1405](https://github.com/stripe/stripe-java/pull/1405) chore: Update gson version in README.
