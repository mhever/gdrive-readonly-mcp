# Review 007: Search Bug Fix

**Date:** 2026-04-01
**Scope:** drive.go (searchFiles), tools.go (description + jsonschema), drive_test.go
**Reviewers:** Opus, Codex (adversarial)

## Bug

`gdrive_search` used `name contains 'X' or fullText contains 'X'` which mixed filename and content results. Searching for "tracker" returned Kubernetes books instead of the file named "tracker".

## Fix

Search by filename first. If no results and not paginating, fall back to full-text content search.

## Findings

| ID | Severity | Found By | Summary | Status |
|----|----------|----------|---------|--------|
| 1 | Critical | Both | Fallback nextPageToken would be misrouted to name query on next page | Fixed — clear NextPageToken on fallback results |
| 2 | Major | Codex | Any name match suppresses content matches | By design — documented in tool description |
| 3 | Major | Both | No test coverage for two-phase logic | Deferred — requires mock Drive service |
| 4 | Minor | Opus | searchInput.Query jsonschema tag didn't reflect name-first behavior | Fixed |
| 5 | Minor | Codex | Double API call latency on fallback path | Accepted — only on no-name-match cases |

## Design Decision

Name matches suppressing content matches (#2) is the intended behavior per the original bug report. The user specifically wanted filename matches prioritized. Content fallback is single-page only to avoid pageToken cross-contamination.
