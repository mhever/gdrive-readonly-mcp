# Review 006: Timeouts + Rate Limiting

**Date:** 2026-03-31
**Scope:** timeout.go, timeout_test.go, drive.go, docs.go, sheets.go, tools.go
**Reviewers:** Opus, Codex (adversarial)

## Findings

| ID | Severity | Found By | Summary | Status |
|----|----------|----------|---------|--------|
| M1 | Major | Both | Rate limiter + timeout at handler layer, not per-API-call | Fixed — moved both to API-calling functions |
| M2 | Major | Both | `downloadFile` shares one timeout across metadata + download | Fixed — separate timeout per call |
| M3 | Major | Both | `readSpreadsheet` shares one timeout across metadata + values | Fixed — separate timeout per call |
| M4 | Major | Both | `handleReadFile` double-fetches metadata | Fixed — `downloadFile` accepts pre-fetched metadata |
| M5 | Major | Both | Tests are shallow / test stdlib not project code | Fixed — removed stdlib tests, added real rate limit test |
| m1 | Minor | Codex | Nested timeouts may silently shorten inner operations | Resolved — per-call timeouts eliminate nesting |
| m2 | Minor | Codex | 30s may be too short for large Sheets reads | Deferred — documented limitation |
| m3 | Minor | Codex | `apiLimiter` is global with no way to reset for tests | Deferred — acceptable for single-binary |
| c1 | Cosmetic | Opus | README says "outbound API requests" | Fixed — changed to "Google API calls" |

## Key Theme

Both reviewers independently concluded: move rate limiting and per-call timeouts INTO the API-calling functions (drive.go, docs.go, sheets.go), not the handlers. This fixes M1-M4 systematically.
