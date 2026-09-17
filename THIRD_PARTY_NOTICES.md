# Third-Party Notices

## DEEIX Chat

Ruile's billing foundation contains a modified, Ruile-specific implementation
of integer pricing arithmetic and billing-domain separation informed by
DEEIX-Chat.

- Project: DEEIX Chat
- Source: https://github.com/DEEIX-AI/DEEIX-Chat
- License: Apache License 2.0
- Copyright: Copyright 2026 DEEIX-AI

The adapted code is isolated under `internal/billing/` and carries modification
notices. Ruile does not copy DEEIX-Chat's user/account ownership model; all
billing records are keyed by Ruile workspace (`tenant_id`).
