# Unifies the billing cycle anchor format across Subscriptions and Invoices

## What’s new

Unifies how updates to a subscription’s billing cycle anchor are formatted when [updating Subscriptions](https://docs.stripe.com/api/subscriptions/update.md?api-version=2026-09-30.endive#update_subscription-billing_cycle_anchor), [resuming Subscriptions](https://docs.stripe.com/api/subscriptions/resume.md?api-version=2026-09-30.endive#resume_subscription-billing_cycle_anchor), and [creating Preview Invoices](https://docs.stripe.com/api/invoices/create_preview.md?api-version=2026-09-30.endive#create_invoice-subscription_details-billing_cycle_anchor). The [billing_cycle_anchor](https://docs.stripe.com/api/subscriptions/update.md?api-version=2026-09-30.endive#update_subscription-billing_cycle_anchor) field now uses a consistent object format with a `type` field, replacing the previous string, enum, and timestamp formats.

## Changes

#### REST API

| Fields | Change | From → to |
| --- | --- | --- |
| `Invoice#create_preview.subscription_details.billing_cycle_anchor` | Changed | `enum('now'|'unchanged') | DateTime → billing_cycle_anchor_param` |
| `Subscription#resume.billing_cycle_anchor` | Changed | `enum('now'|'unchanged') → billing_cycle_anchor_param` |
| `Subscription#update.billing_cycle_anchor` | Changed | `enum('now'|'unchanged') → billing_cycle_anchor_param` |
