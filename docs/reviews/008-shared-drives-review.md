# Review 008: Shared Drives Fix

**Date:** 2026-04-01
**Scope:** drive.go — shared drives support on all API calls
**Reviewers:** Opus, Codex (adversarial)

## Findings

| ID | Severity | Found By | Summary | Status |
|----|----------|----------|---------|--------|
| 1 | Critical | Codex | `Corpora("allDrives")` skipped when folderID set — shared drive folders won't work | Fixed — always set |
| 2 | Major | Codex | `result.IncompleteSearch` never checked — silently partial results | Fixed — warning in output |
| 3 | Minor | Codex | `allDrives` higher latency than narrower corpora | Accepted — trade-off for "just works" |

## Notes

- Opus said the Corpora conditional was correct; Codex disagreed. Drive API docs were ambiguous. Went with Codex's safer approach (always set allDrives).
- `IncludeItemsFromAllDrives` does NOT bypass ACLs — only returns files the user has permission to access.
- `SupportsAllDrives` on `Files.Get` confirmed sufficient for both metadata and download.
