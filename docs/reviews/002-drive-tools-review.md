# Review 002: Drive Tools (Step 3)

**Date:** 2026-03-31
**Scope:** drive.go, drive_test.go, tools.go, tools_test.go, main.go (updated)
**Reviewers:** Opus, Codex (adversarial)

## Findings

| ID | Severity | Found By | Summary | Status |
|----|----------|----------|---------|--------|
| M1 | Major | Both | `wrapAPIError` discards original error — no `%w` wrapping for 404/403/429 | Fixed — added `%w` wrapping |
| M2 | Major | Opus | Env vars diverge from plan: `GDRIVE_CREDENTIALS_PATH`/`GDRIVE_TOKEN_PATH` vs `CREDENTIALS_FILE`/`TOKEN_FILE` | Fixed — changed to match plan |
| M3 | Major | Both | `downloadFile` has zero test coverage — most security-critical function | Fixed — added ID validation tests |
| M4 | Major | Codex | `downloadFile` TOCTOU between metadata check and download (mitigated by LimitReader) | Fixed — added documentation comment |
| m1 | Minor | Both | `pageToken` passed to API without any validation (no length cap) | Fixed — added 500 char cap |
| m2 | Minor | Opus | `searchFiles` hardcodes PageSize(20), no pagination, but returns nextPageToken field | Fixed — added page_size/page_token params |
| m3 | Minor | Both | `isTextMime` missing common types: yaml, markdown, toml, sh, sql | Fixed — added 10 MIME types |
| m4 | Minor | Opus | `formatFileList` says "Found N file(s)" but means current page only | Fixed — "Showing" + pagination hint |
| m5 | Minor | Both | `wrapAPIError` parameter named `context` shadows the `context` package | Fixed — renamed to `operation` |
| m6 | Minor | Codex | `formatFileList` doesn't truncate very long file names | Fixed — 500 char cap with "..." |
| c1 | Cosmetic | Opus | Inconsistent `jsonschema` tag style — verify SDK processes them | Skipped |
| c2 | Cosmetic | Codex | `handleGetFileMetadata` redundant empty check before `validateFileID` | Skipped |

## Notes

- Double validation of folderID (handler + drive func) is harmless defense-in-depth
- `downloadFile`/`isTextMime` implemented early (Step 4 scope) — minor scope creep
- Global service vars acceptable for single-binary CLI but noted for testability
- `downloadFile` size limit is properly enforced via LimitReader — cannot be bypassed
- `formatFileList` is safe for practical use (MCP JSON-RPC escapes special chars)

## Verification

- `go test ./...` — all pass
- `go vet ./...` — clean
- `go build` — compiles successfully
