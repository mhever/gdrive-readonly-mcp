# Review 003: Content Reading (Step 4)

**Date:** 2026-03-31
**Scope:** docs.go, docs_test.go, sheets.go, sheets_test.go, tools.go (updated), tools_test.go (updated)
**Reviewers:** Opus, Codex (adversarial)

## Findings

| ID | Severity | Found By | Summary | Status |
|----|----------|----------|---------|--------|
| M1 | Major | Both | TSV injection — embedded tabs/newlines in cells corrupt output structure | Fixed — `sanitizeTSVCell()` escapes delimiters |
| M2 | Major | Codex | CSV formula injection — cells starting with `=+\-@` not sanitized | Fixed — prefix with `'` in sanitizer |
| M3 | Major | Codex | No size limits on Docs/Sheets responses — OOM risk | Fixed — 5MB text cap (docs), 500K cell cap + 5MB TSV cap (sheets) |
| M4 | Major | Opus | `readSheetInput.Range` no length validation | Fixed — 500 char cap |
| m1 | Minor | Opus | Sheet title needs quoting for special chars | Fixed — single-quote wrapping with escaping |
| m2 | Minor | Opus | MIME dispatch test is hollow | Fixed — renamed to honest description |
| m3 | Minor | Opus | `<nil>` rendered for nil cell values | Fixed — empty string for nil |
| m4 | Minor | Codex | Downloads binary files before checking MIME | Fixed — check meta.MimeType first |
| m5 | Minor | Codex | `spreadsheet.Sheets[0].Properties` nil check missing | Fixed — added nil guard |
| c1 | Cosmetic | Both | Unsupported Google Apps error uses uppercase | Skipped |
| c2 | Cosmetic | Opus | TSV output has no trailing newline | Skipped |

## Notes

- Go's nil-slice range semantics make doc tree walking safe (no nil pointer crashes)
- downloadFile size limit properly enforced via LimitReader
- Codex's formula injection finding is excellent — real security concern for a portfolio project
- Size limit asymmetry (1MB for files, unlimited for Docs/Sheets) is a valid concern
