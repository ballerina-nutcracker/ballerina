# Go/JVM behavioural-parity risk areas

Divergence hot-spots to investigate when checking a Go-native stdlib against the jBallerina (JVM) reference. Used by `add-stdlib-support` Step 5 and `fill-stdlib-gap` Step 3.

## Areas to investigate for every stdlib

- **Decimal/floating-point precision** — Ballerina `decimal` maps to `java.math.BigDecimal` on the JVM. Verify Go preserves the same precision, rounding mode, and string representation.
- **String encoding** — Java uses UTF-16 internally; Go uses UTF-8. Check whether string operations (length, indexing, formatting) can produce different output for non-ASCII input.
- **Error messages** — Differences in the *underlying* exception/error text between Java and Go are **acceptable**. The **outer Ballerina error message and error type** must stay consistent; the text of `error:Cause` (the raw Java/Go error) may diverge.
- **Numeric overflow and edge cases** — Verify min/max values, overflow semantics, and NaN/Infinity handling against the jBallerina reference.
- **Module-specific risks** — each domain has its own divergence hot-spots; the time-module list below shows the expected depth.

## Common jBallerina defect patterns

A parity check compares against jBallerina, but jBallerina itself has bugs that the port must not copy (see `add-stdlib-support` Step 5). When reading the Java natives, look for these patterns and probe them with **`run-jballerina`**:

- **Narrowing a Ballerina `int`** — `(int) obj.getIntValue(...)` silently truncates a 64-bit value to 32 bits, so large inputs behave as small ones.
- **Narrowing a Ballerina `float`** — `(float) obj.getFloatValue(...)` computes in 32-bit precision, changing rounding and overflowing earlier than the 64-bit `float` the user passed.
- **Truncating a `decimal`** — `BigDecimal.longValue()` drops the fraction and wraps on overflow instead of failing, e.g. comparing times at whole-second granularity.
- **Static mutable state in natives** — a `static` field that per-instance calls reassign is shared across all instances and races between strands.
- **Check-then-act across calls** — a `.bal` method that checks with one extern and acts with another is not atomic under concurrency.
- **Limits checked in the wrong order** — a capacity or size check that runs before the existing-key branch, or that can compute zero, lets the limit be exceeded or evicts more than needed.
- **Inputs ignored without an error** — out-of-range arguments silently replaced by a default instead of being rejected or documented.

## Domain-specific risks (time module example)

- RFC 3339 / RFC 5322 parsing edge cases (trailing spaces, lowercase `z`, sub-second precision beyond 9 digits).
- `utcToEmailString` zone representation (e.g., `"0"` → `"GMT"` in jBallerina).
- Sub-second precision in `utcToString` / `civilToString`.
- Leap second handling — Java's `java.time` and Go's `time` package model these differently.
- Timezone data source — Java ships IANA zone DB; Go uses OS-supplied or embedded `tzdata`.
- `monotonicNow()` epoch — explicitly "unspecified epoch"; a divergence here is acceptable, document it.

## How to verify a parity row

Use the **`run-jballerina`** skill: write a small `.bal` probe exercising the behaviour, run it with `bal run` (jBallerina) and with `go run ./cli/cmd run` (this interpreter), and compare stdout, stderr, exit status, and error/panic line numbers.
