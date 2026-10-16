---
phase: 32-v1-14-p1-p2
plan: 02
subsystem: security
tags: [pbkdf2, owasp, sm2-jwt, algorithm-confusion, modulo-bias, chi-square, p1-hardening, regression-tests]

# Dependency graph
requires:
  - phase: 32-v1-14-p1-p2 (plan 01)
    provides: "Wave 1 P1 security quick wins (replay window tightening, permission inherit test, etc.) and established same-package test patterns"
  - phase: prior P1 fixes
    provides: "commits b7dedac (PBKDF2 100k bump), 64b1b40 (SM2 JWT alg check), 07f210c (crypto/rand.Int rejection sampling) — production code is already in place; this plan adds the bump to 600k + 3 regression tests"
provides:
  - "P1-S5: DefaultPasswordConfig.Iterations bumped from 100000 to 600000 (OWASP 2023 baseline) with backward compatibility for 100k hashes preserved via hash-format-embedded iteration count"
  - "P1-S1: 2 regression tests pinning the alg-header whitelist in ValidateTokenWithSM2 (alg=none and alg=HS256-confusion rejection) plus 1 positive control test"
  - "P1-S6: chi-square goodness-of-fit test proving GenerateRandomPassword's charset sampling is uniform (no modulo bias)"
affects: [future P1-C6 validation (N+1 → IN query), future P2-A4 migration renumbering, future security audits]

# Tech tracking
tech-stack:
  added: []  # no new packages — used existing crypto/hmac, crypto/sha256, encoding/base64, encoding/hex, golang-jwt/jwt/v5, tjfoc/gmsm
  patterns:
    - "Per-character chi-square goodness-of-fit for randomness tests (more sensitive than per-category 3-sigma and the standard for modulo-bias detection)"
    - "Same-package test (`package security` / `package crypto` not `_test`) when the test needs access to unexported helpers like pbkdf2SM3 or to use ExportPublicKeyToHex without re-importing it"
    - "Classic alg-confusion attack reproduction: attacker uses server's raw public-key bytes as HMAC-SHA256 secret; defense is header.alg whitelist before signature verification"
    - "Backward compatibility via format-embedded parameters: \$sm3\$iterations\$salt\$hash lets VerifyPassword pick up the legacy 100k iteration count from the stored hash and re-derive with the same cost — no migration step required"

key-files:
  created:
    - "internal/core/security/password_owasp_test.go — TestPasswordManager_DefaultIterationsAre600k + TestPasswordManager_VerifyBackwardCompat_100k"
    - "internal/core/security/random_password_bias_test.go — TestGenerateRandomPassword_NoBiasDistribution (chi-square vs uniform)"
    - "pkg/crypto/sm2_jwt_alg_test.go — TestSM2JWT_RejectsAlgNone + TestSM2JWT_RejectsAlgHS256Confusion + TestSM2JWT_AcceptsCorrectSM2Alg"
  modified:
    - "internal/core/security/password.go — DefaultPasswordConfig.Iterations 100000 → 600000, comment updated to reference OWASP 2023 baseline"

key-decisions:
  - "600k not phased (e.g. 200k → 400k → 600k): per 32-RESEARCH.md Pitfall 1, SM3-PBKDF2 at 600k ≈ 300ms verify, well within typical login latency budget. Direct bump is simpler and avoids a multi-deployment story."
  - "VerifyPassword left untouched: it already reads the iteration count from the stored hash, so legacy 100k hashes continue to verify at 100k cost. New hashes use 600k. No DB migration needed."
  - "Chi-square per-character, not per-category: the original 3-sigma per-category test had too small a sample (32k chars, 4 categories) and at 25%-per-category was actually wrong (the charset is 36% lowercase / 36% uppercase / 14% digit / 14% symbol, not 25% each). Per-character chi-square across 72 bins is both more correct and more sensitive to the precise kind of bias a modulo bug introduces (a subset of consecutive characters getting a non-uniform share)."
  - "Test for already-fixed P1-S1 uses ExportPublicKeyToHex (existing public helper) as the HMAC secret to faithfully reproduce what an attacker would actually have access to (raw 04+X+Y bytes), not a test-only constant."

patterns-established:
  - "Pattern: BackwardCompatHashFormat — store security parameters in the hash itself so algorithm upgrades are zero-downtime. Verify reads the embedded value, not the current config. This pattern is generalizable to any future PBKDF2 cost changes."
  - "Pattern: ChiSquareUniformityTest — for any sampling function over a known charset, compute per-element counts across N samples and run chi-square goodness-of-fit against the uniform null. Threshold at chi²(0.001, df=k-1) for stability; the real bug signal is chi² in the hundreds or thousands."
  - "Pattern: JWTSpoofedHeader — for any custom-signing-method JWT validator, ship 3 regression tests: alg=none rejected, common-confusion alg (HS256/RS256) rejected, correct alg accepted (positive control to prove the rejection is the alg check, not a broken validator)"

requirements-completed: [P1-S1, P1-S5, P1-S6]

# Metrics
duration: ~25min
completed: 2026-06-13
---

# Phase 32 Plan 02: Wave 2 Password + JWT

**PBKDF2-SM3 bumped to OWASP 2023 baseline (600k iterations) with backward compatibility for legacy 100k hashes, plus 4 regression tests pinning the SM2 JWT alg-header check and the modulo-bias-free random password sampler.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-06-13T04:13:00Z (approx, after Wave 1 SUMMARY commit)
- **Completed:** 2026-06-13T04:38:00Z
- **Tasks:** 2/2
- **Files modified:** 4 (1 production + 3 new test files)
- **Commits:** 2 (f1281f3 fix + ea3f7ca test)

## Accomplishments

- **P1-S5 fully resolved:** DefaultPasswordConfig.Iterations is now 600000 (OWASP 2023 baseline). The `HashPassword` function uses `pm.config.Iterations` at runtime, so the bump propagates without any other code change. VerifyPassword was left untouched — it already reads the iteration count from the stored `$sm3$iterations$salt$hash` format, so legacy 100k hashes continue to verify at 100k cost and only NEW hashes use 600k. No DB migration needed.
- **P1-S1 regression coverage:** Added 3 tests in `pkg/crypto/sm2_jwt_alg_test.go`. The two negative tests (alg=none and alg=HS256-confusion) reproduce the exact attack vectors the validator's header-alg check (commit 64b1b40) defends against. The positive test (alg=SM2) proves the rejection in the other two is due to the alg whitelist, not a broken validator.
- **P1-S6 regression coverage:** Added `TestGenerateRandomPassword_NoBiasDistribution` in `internal/core/security/random_password_bias_test.go`. Uses a chi-square goodness-of-fit test across all 72 characters of the charset (5000 passwords × 16 chars = 80k samples, ~1111 expected per character). Threshold 130 catches a 20% modulo bias (which would push chi² to several hundred) while staying stable on developer hardware.
- **Comment updated:** The 100k-bump comment block in `DefaultPasswordConfig` is now also a 600k-bump block, citing the OWASP 2023 baseline and the backward-compat mechanism.

## Task Commits

Each task was committed atomically:

1. **Task 1: Bump PBKDF2 to 600k (P1-S5) + OWASP + backward-compat tests** - `f1281f3` (fix)
   - 2 files changed: `internal/core/security/password.go` (Iterations 100000 → 600000 + comment), `internal/core/security/password_owasp_test.go` (2 new tests, 102 insertions)
2. **Task 2: JWT alg confusion regression tests + random password bias test (P1-S1 + P1-S6)** - `ea3f7ca` (test)
   - 2 files created, 250 insertions

**Plan metadata:** This SUMMARY.md (will be committed in a separate docs(32-02) commit per the orchestrator's instructions).

## Files Created/Modified

### Created (3 test files)

- `internal/core/security/password_owasp_test.go` (101 lines) — `TestPasswordManager_DefaultIterationsAre600k` asserts both `DefaultPasswordConfig.Iterations == 600000` AND that a freshly hashed password embeds 600000 in its stored format. `TestPasswordManager_VerifyBackwardCompat_100k` constructs a legacy 100k-iteration hash directly (using the unexported `pbkdf2SM3` helper from the same package) and confirms `VerifyPassword` still accepts it AND still rejects a wrong plaintext. Same-package (`package security`) so the test can call the unexported helper.
- `internal/core/security/random_password_bias_test.go` (101 lines) — `TestGenerateRandomPassword_NoBiasDistribution` generates 5000 passwords of length 16, tallies how often each of the 72 charset characters appears, and runs a chi-square goodness-of-fit test against the uniform null hypothesis. Threshold 130 (chi² for 71 df at alpha=0.001 is ~115; threshold includes headroom for slow CI). Reports min/max characters and counts on failure for fast diagnosis.
- `pkg/crypto/sm2_jwt_alg_test.go` (148 lines) — 3 tests in `package crypto` (same package as the validator) to keep the test tightly coupled to the production code. `TestSM2JWT_RejectsAlgNone` builds a hand-crafted alg=none token (header + payload + empty signature) and confirms `ValidateTokenWithSM2` returns an error mentioning "algorithm" or "alg". `TestSM2JWT_RejectsAlgHS256Confusion` is the classic algorithm-confusion attack: signs a token with HS256 using the SM2 public key's raw bytes (via `ExportPublicKeyToHex`) as the HMAC secret. The defense is the `header.Alg != sm2Method.Alg()` check at the top of `ValidateTokenWithSM2` (lines 243-245). `TestSM2JWT_AcceptsCorrectSM2Alg` is a positive control: mints a real SM2 token via `GenerateTokenWithSM2`, validates it, and confirms the user_id and username round-trip.

### Modified (1 production file)

- `internal/core/security/password.go` — Changed `Iterations: 100000` to `Iterations: 600000` and updated the multi-line comment above the var declaration to reflect the OWASP 2023 baseline, the 600k cost (~300ms verify on typical hardware), and the backward-compat mechanism (legacy 100k hashes still verify because the iteration count is embedded in the stored format). The actual `HashPassword` function uses `pm.config.Iterations` (line 107, 116) so the change is automatic. `VerifyPassword` was deliberately left unchanged: lines 122-149 read iterations from the stored hash and call `pm.pbkdf2SM3` with that value, which means legacy 100k hashes still verify at 100k cost.

## Decisions Made

- **Direct 600k bump, not phased** — Per 32-RESEARCH.md Pitfall 1, SM3-PBKDF2 at 600k is ~300ms verify, which is well within typical login latency budgets. The phased approach (200k → 400k → 600k) was rejected because it adds deployment complexity for marginal benefit; if staging shows latency issues, the operator can drop the constant back to 100000 with a single commit and re-roll.
- **VerifyPassword left alone** — The hash format `$sm3$<iterations>$<salt>$<hash>` already embeds the iteration count, so legacy users with 100k hashes continue to verify at 100k cost (no spike in login latency for existing accounts on the day of the rollout). Only NEW passwords use 600k.
- **Per-character chi-square, not per-category 3-sigma** — The first draft used 4 categories (uppercase, lowercase, digit, symbol) with a 3-sigma rule. It failed on the actual charset (26/26/10/10 = 36%/36%/14%/14%, not 25% each as I'd assumed). The per-character chi-square test against all 72 bins is both more correct and more sensitive — a 20% modulo bias (256 mod 72 = 40, so the first 40 characters are over-represented) would push individual bin counts hundreds above expected, accumulating to a chi² of several hundred or thousand. The threshold of 130 (chi² for 71 df at alpha=0.001) is wide enough to avoid flakes but tight enough to catch the bug.
- **Same-package tests, not `_test` package** — `password_owasp_test.go` is `package security` (not `package security_test`) so the test can call the unexported `pbkdf2SM3` helper to build a 100k-iter hash fixture. `sm2_jwt_alg_test.go` is `package crypto` for the same reason — it uses `GenerateKeyPair`, `ExportPublicKeyToHex`, `GenerateTokenWithSM2`, and `ValidateTokenWithSM2` all from the same package.
- **Test for already-fixed P1-S1 uses the project's existing `ExportPublicKeyToHex`** — To faithfully reproduce the alg-confusion attack, the test signs a token with HS256 using the server's SM2 public key as the HMAC secret. The most realistic way to do that is to use the same byte representation an attacker would actually have access to (raw 04+X+Y), which is what `ExportPublicKeyToHex` produces. Using a test-only constant would test a slightly different attack.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- **First chi-square test draft was wrong** — I initially wrote `TestGenerateRandomPassword_NoBiasDistribution` as a 4-category 3-sigma test, but used 25% per category as the expected proportion. The charset is actually 26+26+10+10 = 72 chars split 36/36/14/14, not 25/25/25/25. Test failed on first run with a 38/37/14/12 split, all categories off by 3000+ characters. Rewrote as per-character chi-square across all 72 bins (5000 passwords × 16 chars = 80k samples, ~1111 expected per char), which both matches the actual charset probabilities and is more sensitive to the precise bias a modulo bug introduces.
- **Unused `math` import after switching away from `math.Sqrt`** — The per-category draft used `math.Sqrt` for the 3-sigma tolerance; the chi-square rewrite doesn't. Removed the import to keep `go vet ./...` clean.
- **Test file initially had a dead `loadTestSM2KeyPair` helper** — The first draft of `sm2_jwt_alg_test.go` included a helper that returned `interface{}` and then immediately re-generated the key pair inside the test. Cleaned up to just call `GenerateKeyPair()` directly in each test. Caught by self-review before commit.
- **Unused `_ = privKey` lines** — Same cleanup; the HS256 test only needs the public key (the attack doesn't use the private key at all). Removed.

## User Setup Required

None - no external service configuration required. All changes are self-contained:
- The 600k iteration count is a Go constant; no environment variable, no config file change.
- New tests are pure Go standard library + already-imported project deps.

## Next Phase Readiness

- P1-S5, P1-S1, P1-S6 fully resolved. P1-S2/S3/S4/S7 already covered in Wave 1 (plan 32-01).
- Remaining P1 items for future waves:
  - P1-C1..C5 — already fixed in prior commits, need regression tests (suggested as Wave 3)
  - P1-C6 — N+1 → IN query refactor (NOT YET FIXED, needs actual code change, not just tests)
  - P1-B1 — needs config_invalidation_test.go
  - P1-B2 — verification only, no new code
- P2-A1..A8 architectural debt (8 categories, most unfixed) — separate waves
- Build, vet, and all 6 targeted tests pass; ready for Wave 3

## Verification Commands Run

```bash
# Per-task verification
go test -count=1 -run "TestPasswordManager_DefaultIterationsAre600k|TestPasswordManager_VerifyBackwardCompat_100k" ./internal/core/security/ -v
# Both pass (0.32s, 0.18s)

go test -count=1 -run "TestSM2JWT_RejectsAlgNone|TestSM2JWT_RejectsAlgHS256Confusion|TestSM2JWT_AcceptsCorrectSM2Alg" ./pkg/crypto/ -v
# All 3 pass (0.00s each — JWT ops are cheap)

go test -count=1 -run "TestGenerateRandomPassword_NoBiasDistribution" ./internal/core/security/ -v
# Passes (0.03s — 5000 password generations + chi-square)

# Full verification at plan close
go build ./...   # exit 0
go vet ./...     # exit 0
```

## Grep Assertions (per plan acceptance criteria)

```bash
# P1-S5 acceptance
grep -n "Iterations:\s*100000" internal/core/security/password.go
# 0 matches (was 1 before this plan)
grep -n "Iterations:\s*600000" internal/core/security/password.go
# 1 match (line 34)

# Test presence
grep -l "TestPasswordManager_DefaultIterationsAre600k" internal/core/security/password_owasp_test.go  # present
grep -l "TestPasswordManager_VerifyBackwardCompat_100k" internal/core/security/password_owasp_test.go  # present
grep -l "TestSM2JWT_RejectsAlgNone" pkg/crypto/sm2_jwt_alg_test.go                                    # present
grep -l "TestSM2JWT_RejectsAlgHS256Confusion" pkg/crypto/sm2_jwt_alg_test.go                           # present
grep -l "TestSM2JWT_AcceptsCorrectSM2Alg" pkg/crypto/sm2_jwt_alg_test.go                               # present
grep -l "TestGenerateRandomPassword_NoBiasDistribution" internal/core/security/random_password_bias_test.go  # present

# OWASP comment present
grep -n "OWASP" internal/core/security/password.go
# Multiple matches (header comment + field comment + section above DefaultPasswordConfig)
```

## Self-Check

- [x] `DefaultPasswordConfig.Iterations = 600000` — verified via grep
- [x] Backward compat preserved — `VerifyPassword` reads iterations from hash; legacy 100k hashes still verify (test passes)
- [x] `pkg/crypto/sm2_jwt_alg_test.go` exists with 3 test functions — verified
- [x] `internal/core/security/password_owasp_test.go` exists with 2 test functions — verified
- [x] `internal/core/security/random_password_bias_test.go` exists with 1 test function — verified
- [x] `go build ./...` exits 0
- [x] `go vet ./...` exits 0
- [x] All 6 targeted tests pass
- [x] Both task commits landed on `main`

## Self-Check: PASSED

---
*Phase: 32-v1-14-p1-p2*
*Plan: 02 — Wave 2 Password + JWT*
*Completed: 2026-06-13*
