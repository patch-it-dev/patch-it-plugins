# Removes the payment method types parameter from Payment Intents and Setup Intents

## What’s new

Removes `payment_method_types` as a writable parameter from the [create](https://docs.stripe.com/api/payment_intents/create.md?api-version=2026-09-30.endive), [update](https://docs.stripe.com/api/payment_intents/update.md?api-version=2026-09-30.endive), and [confirm](https://docs.stripe.com/api/payment_intents/confirm.md?api-version=2026-09-30.endive) methods on Payment Intents, and from the [create](https://docs.stripe.com/api/setup_intents/create.md?api-version=2026-09-30.endive) and [update](https://docs.stripe.com/api/setup_intents/update.md?api-version=2026-09-30.endive) methods on Setup Intents. Passing `payment_method_types` on this API version returns a 400 error with the code `payment_method_types_no_longer_supported`.

Use [dynamic payment methods](https://docs.stripe.com/payments/payment-methods/dynamic-payment-methods.md) to automatically determine available payment methods based on other parameters such as `currency`, `amount`, and `customer`. If you need more control:

- Use [excluded_payment_method_types](https://docs.stripe.com/api/payment_intents/create.md?api-version=2026-09-30.endive#create_payment_intent-excluded_payment_method_types) to exclude specific types from the set determined by dynamic payment methods.
- Use [allowed_payment_method_types](https://docs.stripe.com/api/payment_intents/create.md?api-version=2026-09-30.endive#create_payment_intent-allowed_payment_method_types) to specify a list of types you want to accept. Stripe filters out incompatible types instead of returning an error.

`payment_method_types` remains as a read-only property on the [Payment Intent](https://docs.stripe.com/api/payment_intents/object.md?api-version=2026-09-30.endive#payment_intent_object-payment_method_types) and [Setup Intent](https://docs.stripe.com/api/setup_intents/object.md?api-version=2026-09-30.endive#setup_intent_object-payment_method_types) objects, reflecting the computed list of available payment methods.

## Changes

#### REST API

| Parameter | Change | Resources or endpoints |
| --- | --- | --- |
| `payment_method_types` | Removed | [PaymentIntent#confirm](/api/payment_intents/confirm?api-version=2026-09-30.endive#confirm_payment_intent), [PaymentIntent#create](/api/payment_intents/create?api-version=2026-09-30.endive#create_payment_intent), [PaymentIntent#update](/api/payment_intents/update?api-version=2026-09-30.endive#update_payment_intent), [SetupIntent#create](/api/setup_intents/create?api-version=2026-09-30.endive#create_setup_intent), [SetupIntent#update](/api/setup_intents/update?api-version=2026-09-30.endive#update_setup_intent) |
