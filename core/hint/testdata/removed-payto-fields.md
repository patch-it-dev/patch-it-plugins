# Removes PayTo-specific fields from Payment Methods

## What’s new

Removes the [payto](https://docs.stripe.com/api/payment_methods/update.md?api-version=2026-09-30.endive) field (containing [account_number](https://docs.stripe.com/api/payment_methods/update.md?api-version=2026-09-30.endive), [bsb_number](https://docs.stripe.com/api/payment_methods/update.md?api-version=2026-09-30.endive), and [pay_id](https://docs.stripe.com/api/payment_methods/update.md?api-version=2026-09-30.endive)) when you [update PaymentMethods](https://docs.stripe.com/api/payment_methods/update.md?api-version=2026-09-30.endive). Previously, the API accepted these [PayTo](https://docs.stripe.com/payments/payto.md)-specific fields with a `200` status but never persisted the values. You can still update other fields such as [billing_details](https://docs.stripe.com/api/payment_methods/update.md?api-version=2026-09-30.endive#update_payment_method-billing_details) and [metadata](https://docs.stripe.com/api/payment_methods/update.md?api-version=2026-09-30.endive#update_payment_method-metadata) for PayTo Payment Methods.

## Changes

#### REST API

| Parameter | Change | Resources or endpoints |
| --- | --- | --- |
| `payto` | Removed | [PaymentMethod#update](/api/payment_methods/update?api-version=2026-09-30.endive#update_payment_method) |
