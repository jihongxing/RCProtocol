# PostgreSQL init notes

- `001_init.sql` is the authoritative bootstrap schema for local scaffold startup.
- `002_seed.sql` provides deterministic assets for local end-to-end flow testing.
- Protocol truth stays in `assets`, `asset_state_events`, and `idempotency_records`.
- If you need a clean rerun, recreate the local PostgreSQL volume/database so init scripts execute again.
- Governance-side tables should be added later without duplicating protocol truth.
