# DEEIX-Chat Billing Adaptation

## Source

- Repository: https://github.com/DEEIX-AI/DEEIX-Chat
- License: Apache License 2.0
- Upstream notice: `DEEIX Chat`, Copyright 2026 DEEIX
- Reviewed branch: `dev`
- Review date: 2026-09-16

## Adapted Scope

Ruile only adopts the reusable billing ideas needed by the accepted billing
foundation and step 2A:

- integer-only monetary and point arithmetic;
- fixed-point multiplier composition;
- explicit separation between upstream base cost and billed user points;
- immutable pricing snapshots on subscription and future usage records;
- model-price version resolution;
- usage reservation, release, idempotent settlement, and reconciliation states;
- tests for rounding and multiplier composition.

The implementation in `internal/billing/pricing` is modified for Ruile and is
not a verbatim copy. Ruile uses `tenant_id` as the billing owner instead of the
upstream account identity model.

## Excluded Scope

Step 2A still excludes payment providers, order flows, service-item pricing,
enterprise member allocations, storage charging, and non-chat AI call chains.
Runtime billing remains default-off; observe mode records but never blocks or
debits existing users.
