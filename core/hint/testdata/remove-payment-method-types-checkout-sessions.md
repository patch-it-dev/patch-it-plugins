# Removes the payment method types parameter from Checkout Sessions

## What’s new

Removes the [payment_method_types](https://docs.stripe.com/api/checkout/sessions/create.md?api-version=2026-09-30.endive#create_checkout_session-payment_method_types) parameter from the [create Checkout Session](https://docs.stripe.com/api/checkout/sessions/create.md?api-version=2026-09-30.endive) method. You can no longer pass `payment_method_types` when you create a Checkout Session. Use [dynamic payment methods](https://docs.stripe.com/payments/payment-methods/dynamic-payment-methods.md) to manage payment methods from the Dashboard without writing code.

You can also use [excluded_payment_method_types](https://docs.stripe.com/api/checkout/sessions/create.md?api-version=2026-09-30.endive#create_checkout_session-excluded_payment_method_types) to exclude specific payment methods from the dynamic payment methods results or the new [allowed_payment_method_types](https://docs.stripe.com/api/checkout/sessions/create.md?api-version=2026-09-30.endive#create_checkout_session-allowed_payment_method_types) parameter to specify a set of payment methods.

The [allowed_payment_method_types](https://docs.stripe.com/api/checkout/sessions/create.md?api-version=2026-09-30.endive#create_checkout_session-allowed_payment_method_types)  parameter accepts an array of payment method type and filters the list to only compatible payment methods based on other parameters such as `currency`, `amount`, and `customer`. The compatible subset is shown in the [payment_method_types](https://docs.stripe.com/api/checkout/sessions/object.md?api-version=2026-09-30.endive#checkout_session_object-payment_method_types) property of the response.

Previously, [payment_method_types](https://docs.stripe.com/api/checkout/sessions/create.md?api-version=2026-08-26.dahlia#create_checkout_session-payment_method_types) returned an error when you specified an incompatible payment method. `allowed_payment_method_types` silently filters out incompatible methods and returns only those that work with the current configuration.

## Changes

#### REST API

| Parameters | Change | Resources or endpoints |
| --- | --- | --- |
| `allowed_payment_method_types` | Added | [Checkout.Session](/api/checkout/sessions/object?api-version=2026-09-30.endive#checkout_session_object), [Checkout.Session#create](/api/checkout/sessions/create?api-version=2026-09-30.endive#create_checkout_session) |
| `payment_method_types` | Removed | [Checkout.Session#create](/api/checkout/sessions/create?api-version=2026-09-30.endive#create_checkout_session) |
