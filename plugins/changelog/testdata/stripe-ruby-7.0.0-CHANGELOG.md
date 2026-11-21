## 7.0.0 - 2022-08-02

Breaking changes that arose during code generation of the library that we postponed for the next major version. For changes to the SDK, read more detailed description at https://github.com/stripe/stripe-ruby/wiki/Migration-guide-for-v7. For changes to the Stripe products, read more at https://stripe.com/docs/upgrades#2022-08-01.

"⚠️" symbol highlights breaking changes.

* [#1106](https://github.com/stripe/stripe-ruby/pull/1106) API Updates
* [#1092](https://github.com/stripe/stripe-ruby/pull/1092) API Updates
* [#1090](https://github.com/stripe/stripe-ruby/pull/1090) Use auto-generation for `Invoice` methods
* [#1103](https://github.com/stripe/stripe-ruby/pull/1103) Next major release changes

### ⚠️ Changed
* `retrieve_cash_balance` and `update_cash_balance` methods on `Customer` resource no longer requires the second argument to always be `nil`. The methods now now take in `customer_id`, `params`, and `opts` parameters.
* Update default bundle of CA certificates to April 26, 2022.

### Deprecated
* Deprecate `delete` method on `Subscription` resource. Please use `cancel` method instead.

### ⚠️ Removed
* Remove `details` method from `Issuing.Card` resource. The method was not supported.
* Remove `Issuing.CardDetails` resource. Read more at https://stripe.com/docs/issuing/cards/virtual.
* Remove `create` method from `ReportType` resource. The method was not supported.
* Remove `usage_record_summaries` method from `SubscriptionItem` resource. Please use `list_usage_record_summaries` method instead.
* Remove `AlipayAccount`, `BitcoinReceiver`, `BitcoinTransaction`, `Issuing::CardDetails`, `Recipient`, ` RecipientTransfer`, and `ThreeDSecure` resources. The resources were deprecated or no longer in use.
* Remove ability to list `Card` resource for a `Recipient`.
* Remove `cancel` method from `Transfer` resource. The method was deprecated.

## 6.5.0 - 2022-06-29
* [#1084](https://github.com/stripe/stripe-ruby/pull/1084) API Updates
  * Add support for `deliver_card`, `fail_card`, `return_card`, and `ship_card` test helper methods on resource `Issuing.Card`
* [#1076](https://github.com/stripe/stripe-ruby/pull/1076) fix: Update logging to coerce ASCII-8BIT into UTF-8.
