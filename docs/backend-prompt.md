# Backend Autonomous IAM Optimization Prompt

> This file is the source of truth for the backend cron prompt.
> The prompt itself contains instructions for self-improvement (Phase 7).
> Last updated: Round 87 baseline

## Effectiveness Tracking

| Method | Runs | Real Issues Found | False Positives | Keep? |
|--------|------|-------------------|-----------------|-------|
| go build ./... | 14 | 2 (crypto break, ggidcrypto) | 0 | YES |
| go test ./services/... | 14 | 1 (BIOMETRIC_AES_KEY panic) | 0 | YES |
| curl live API endpoints | 3 | 1 (SCIM hardcoded data) | 0 | YES — HIGHEST VALUE |
| grep TODO/stub | 14 | 0 | 100% (HTML attrs, format strings) | NO — REMOVE |
| A→G dimension rotation | 7 cycles | 6 (R74-80), then 0 | 0 after R80 | DEPRECATED |
| Coverage gap targeting | 3 | 0 production risk | 0 | LOW VALUE |

## Key Insight

Live API testing (curl) found real issues that static analysis missed for 14 rounds.
SCIM returning hardcoded mock data was invisible to `go build`, `go test`, and grep.
**Risk-driven live verification > mechanical rotation scanning.**
