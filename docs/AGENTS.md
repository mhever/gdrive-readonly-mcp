# Agent Workflow

This project uses a multi-agent workflow for implementation quality. Every implementation phase follows this pipeline:

## Pipeline

### 1. Coding Agent (Opus)
- Writes the implementation code
- Model: `opus`
- Writes tests alongside code
- Must use latest library versions

### 2. Reviewer Agent (Opus)
- Reviews code from the coding agent
- Model: `opus`
- Focus areas:
  - Security vulnerabilities
  - Test quality — tests must be meaningful, not dummy/placeholder tests
  - Correct use of APIs
  - Error handling completeness
  - Latest dependency versions
- Produces a list of issues classified as: critical, major, minor, cosmetic

### 3. Adversarial Reviewer (Codex)
- Invoked via `/codex:adversarial-review`
- Additional independent review pass
- Looks for issues the opus reviewer may have missed

### 4. Fix Pass
- Fix all critical, major, and minor issues from both reviewers
- Cosmetic issues are skipped
- Re-run tests after fixes

## Rules
- Security is paramount — this is a public portfolio repo
- Never guess when something is unclear — ask the user
- Always use latest versions of libraries, tools, GitHub Actions, etc.
- All OAuth scopes must be read-only — no exceptions
