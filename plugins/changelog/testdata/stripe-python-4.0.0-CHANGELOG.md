## 4.0.0 - 2022-08-02

Breaking changes that arose during code generation of the library that we postponed for the next major version. For changes to the SDK, read more detailed description at https://github.com/stripe/stripe-python/wiki/Migration-guide-for-v4. For changes to the Stripe products, read more at https://stripe.com/docs/upgrades#2022-08-01.

"⚠️" symbol highlights breaking changes.

* [#847](https://github.com/stripe/stripe-python/pull/847) API Updates
* [#845](https://github.com/stripe/stripe-python/pull/845) Next major release changes
* [#836](https://github.com/stripe/stripe-python/pull/836) API Updates. Add Price.create tests.
* [#835](https://github.com/stripe/stripe-python/pull/835) API Updates. Use auto-generation for credit_note and invoice methods.

### ⚠️ Removed
- Removed deprecated `AlipayAccount`, `BitcoinReceiver`, `BitcoinTransaction`, `IssuerFraudRecord`, `Recipient`, `RecipientTransfer`, and  `ThreeDSecure` classes.
- Removed deprecated `Charge.update_dispute` and `Charge.close_dispute` methods that were using legacy REST API endpoint. Prefer [Dispute.modify](https://stripe.com/docs/api/disputes/update?lang=python) and [Dispute.close](https://stripe.com/docs/api/disputes/close?lang=python)
- Removed deprecated `Card.details` method and `CardDetails` resource. The REST API endpoint is not longer supported.
- Removed the deprecated `Source.source_transactions` method. Prefer `SubscriptionItem.list_source_transactions`
- Removed the deprecated `SubscriptionItem.usage_record_summaries` method. Prefer `SubscriptionItem.list_usage_record_summaries`
- Removed the deprecated `Charge.refund` method. Prefer [Refund.create](https://stripe.com/docs/api/refunds/create)

### ⚠️ Changed
- To be consistent with other resource methods, `ApplicationFee.refund` returns an instance of `ApplicationFee` and doesn't mutate the instance of `ApplicationFee`.
- To be consistent with other resource methods, the `Customer.delete_discount` no longer resets the `discount` property to `None` and returns the deleted discount instead. If you were relying on this behavior, reset the discount property manually:
- The `LineItem` resource now inherits from `StripeObject` as it has no methods of it's own.
- To be consistent with other resource methods, the `Subscription.delete_discount` returns an instance of deleted discount and doesn't mutate the instance of `Subscription`.
- Update the CA certificate bundle.
- Request sending logic unified across standard and custom methods (https://github.com/stripe/stripe-python/pull/832)

## 3.5.0 - 2022-06-30
* [#831](https://github.com/stripe/stripe-python/pull/831) API Updates
  * Add support for `deliver_card`, `fail_card`, `return_card`, and `ship_card` test helper methods on resource `Issuing.Card`
  * Switch from using `instance_url` to computing method path in place for custom methods.
  * Switch from using explicit class methods for test helpers instead of using meta-programming.
