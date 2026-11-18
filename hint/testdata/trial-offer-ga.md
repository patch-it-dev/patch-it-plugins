# Makes trial offers generally available

## What’s new

Adds generally available [Trial Offers](https://docs.stripe.com/api/product-catalog/trial-offer.md?api-version=2026-09-30.endive) for subscriptions. Previously, a subscription used a single trial period for the entire subscription. Trial Offers let you define reusable free or paid introductory pricing for individual [subscription items](https://docs.stripe.com/api/subscription_items.md?api-version=2026-09-30.endive), each with its own price and billing period.

A Trial Offer specifies the recurring [price](https://docs.stripe.com/api/product-catalog/trial-offer/object.md?api-version=2026-09-30.endive#product_catalog_trial_offer_object-price) to charge during the trial, the number of billing intervals that the trial lasts, and the [post-trial price](https://docs.stripe.com/api/product-catalog/trial-offer/object.md?api-version=2026-09-30.endive#product_catalog_trial_offer_object-end_behavior-transition-price) to use when the trial ends. Because Trial Offers define the trial price, they support both free and paid trialing periods. When the trial ends, we automatically change the subscription item to the configured post-trial price.

You can apply a trial offer when you create a subscription, add one to an existing subscription item, or replace an item’s existing trial offer. A subscription can contain both trial and non-trial items, each with its own price and billing period. This lets you offer a trial for an add-on or premium product without changing the billing behavior of the subscription’s other items.

Trial offers are reusable across multiple subscriptions. You can’t change their pricing and duration after creation, so [archiving](https://docs.stripe.com/api/product-catalog/trial-offer/object.md?api-version=2026-09-30.endive#product_catalog_trial_offer_object-active) an offer prevents it from being used for new subscription items without affecting trials that are already in progress.

At trial end, you can configure whether Stripe [sends an invoice, cancels, or pauses the subscription](https://docs.stripe.com/api/subscriptions/object.md?api-version=2026-09-30.endive#subscription_object-trial_settings-end_behavior-missing_payment_method) if the customer has no payment method. You can also configure whether the subscription’s [billing cycle anchor](https://docs.stripe.com/api/subscriptions/create.md?api-version=2026-09-30.endive#create_subscription-trial_settings-end_behavior-billing_cycle_anchor) resets when a trial offer ends.

## Changes

#### REST API

|  | Change | Resource |
| --- | --- | --- |
|  | Added | [ProductCatalog.TrialOffer](/api/product-catalog/trial-offer/object?api-version=2026-09-30.endive#product_catalog_trial_offer_object) |

| Endpoints | Change | Resource |
| --- | --- | --- |
| `create`, `list`, `retrieve`, `update` | Added | [ProductCatalog.TrialOffer](/api/product-catalog/trial-offer/object?api-version=2026-09-30.endive#product_catalog_trial_offer_object) |

| Parameters | Change | Resources or endpoints |
| --- | --- | --- |
| `current_trial` | Added | [Invoice#create_preview.subscription_details.items[]](/api/invoices/create_preview?api-version=2026-09-30.endive#create_invoice-subscription_details-items), [Subscription#create.items[]](/api/subscriptions/create?api-version=2026-09-30.endive#create_subscription-items), [Subscription#update.items[]](/api/subscriptions/update?api-version=2026-09-30.endive#update_subscription-items), [SubscriptionItem](/api/subscription_items/object?api-version=2026-09-30.endive#subscription_item_object), [SubscriptionItem#create](/api/subscription_items/create?api-version=2026-09-30.endive#create_subscription_item), [SubscriptionItem#update](/api/subscription_items/update?api-version=2026-09-30.endive#update_subscription_item) |
| `billing_cycle_anchor` | Added | [Subscription#create.trial_settings.end_behavior](/api/subscriptions/create?api-version=2026-09-30.endive#create_subscription-trial_settings-end_behavior), [Subscription#update.trial_settings.end_behavior](/api/subscriptions/update?api-version=2026-09-30.endive#update_subscription-trial_settings-end_behavior), [Subscription.trial_settings.end_behavior](/api/subscriptions/object?api-version=2026-09-30.endive#subscription_object-trial_settings-end_behavior) |

| Fields | Change | From → to |
| --- | --- | --- |
| `Subscription#create.trial_settings.end_behavior.missing_payment_method`, `Subscription#update.trial_settings.end_behavior.missing_payment_method` | Changed | `required → optional` |
