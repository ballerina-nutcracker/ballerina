# Go/JVM behavioural-parity risk areas

Divergence hot-spots to investigate when checking a Go-native stdlib against the jBallerina (JVM) reference. Used by `add-stdlib-support` Step 5 and `fill-stdlib-gap` Step 3.

## Areas to investigate for every stdlib

- **Decimal/floating-point precision** — Ballerina `decimal` maps to `java.math.BigDecimal` on the JVM. Verify Go preserves the same precision, rounding mode, and string representation.
- **String encoding** — Java uses UTF-16 internally; Go uses UTF-8. Check whether string operations (length, indexing, formatting) can produce different output for non-ASCII input.
- **Error messages** — Differences in the *underlying* exception/error text between Java and Go are **acceptable**. The **outer Ballerina error message and error type** must stay consistent; the text of `error:Cause` (the raw Java/Go error) may diverge.
- **Numeric overflow and edge cases** — Verify min/max values, overflow semantics, and NaN/Infinity handling against the jBallerina reference.
- **Module-specific risks** — each domain has its own divergence hot-spots; the time-module list below shows the expected depth.

## Domain-specific risks (time module example)

- RFC 3339 / RFC 5322 parsing edge cases (trailing spaces, lowercase `z`, sub-second precision beyond 9 digits).
- `utcToEmailString` zone representation (e.g., `"0"` → `"GMT"` in jBallerina).
- Sub-second precision in `utcToString` / `civilToString`.
- Leap second handling — Java's `java.time` and Go's `time` package model these differently.
- Timezone data source — Java ships IANA zone DB; Go uses OS-supplied or embedded `tzdata`.
- `monotonicNow()` epoch — explicitly "unspecified epoch"; a divergence here is acceptable, document it.

## How to verify a parity row

Use the **`run-jballerina`** skill: write a small `.bal` probe exercising the behaviour, run it with `bal run` (jBallerina) and with `go run ./cli/cmd run` (this interpreter), and compare stdout, stderr, exit status, and error/panic line numbers.
