# Review 001: Scaffold + Auth (Steps 1–2)

**Date:** 2026-03-31
**Scope:** main.go, auth.go, auth_test.go, tools.go, go.mod, .gitignore
**Reviewers:** Opus, Codex (adversarial)

## Findings

| ID | Severity | Found By | Summary | Status |
|----|----------|----------|---------|--------|
| C1 | Critical | Codex | Committed binary (20.7MB) tracked in git | Verified not tracked — false positive (already in .gitignore) |
| M1 | Major | Both | Race condition in `persistingTokenSource.Token()` — no mutex on `lastToken` | Fixed — added `sync.Mutex` |
| M2 | Major | Codex | TOCTOU on token file permissions — `os.WriteFile` doesn't fix perms on existing files | Fixed — added `os.Chmod(path, 0600)` after write |
| M3 | Major | Codex | OAuth callback handler accepts any HTTP method | Fixed — added GET-only check, returns 405 |
| M4 | Major | Codex | OAuth `error` query param reflected in HTTP response without sanitization | Fixed — added `sanitizeOAuthError()` (truncate + strip control chars) |
| M5 | Major | Both | No tests for `tools.go` — handleStatus and registerTools untested | Fixed — added `tools_test.go` |
| m1 | Minor | Both | `resolveFilePath` calls `log.Fatalf` instead of returning error | Fixed — changed to return `(string, error)` |
| m2 | Minor | Opus | `openBrowser` on Windows vulnerable to shell metachar injection via `cmd /c start` | Fixed — added empty title arg: `start "" <url>` |
| m3 | Minor | Opus | Callback handler goroutine can block on channel send after timeout | Fixed — wrapped sends in `select`/`default` |
| m4 | Minor | Codex | `getTokenFromWeb` mutates caller's `*oauth2.Config` RedirectURL | Fixed — works on a copy of config |
| m5 | Minor | Codex | `openBrowser` spawns zombie process — `cmd.Start()` without `cmd.Wait()` | Fixed — added `go func() { cmd.Wait() }()` |
| m6 | Minor | Codex | Module path `github.com/gdrive-readonly-mcp` missing GitHub owner | Deferred — needs user's GitHub username |
| m7 | Minor | Codex | `escapeQuery` not implemented (promised in CLAUDE.md) | Fixed — added `query.go` + `query_test.go` (11 test cases) |
| c1 | Cosmetic | Opus | `loadOAuthConfig` params could be grouped by type | Skipped |
| c2 | Cosmetic | Opus | Placeholder comment on statusInput/statusOutput | Skipped |
| c3 | Cosmetic | Codex | HTML success page missing DOCTYPE and Content-Type header | Skipped |
| c4 | Cosmetic | Codex | No README.md | Skipped (planned for Step 5) |

## Stats

- **Total findings:** 17 (1 critical, 5 major, 7 minor, 4 cosmetic)
- **Fixed:** 12
- **Skipped (cosmetic):** 4
- **Deferred:** 1 (m6 — module path)
- **False positive:** 1 (C1 — binary was not actually tracked)

## Verification

- `go test ./...` — 19 tests, all pass
- `go vet ./...` — clean
- `go build` — compiles successfully
