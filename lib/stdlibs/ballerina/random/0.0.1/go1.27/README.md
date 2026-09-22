# Ballerina Random Library

## Overview

The `ballerina/random` library provides functions for generating pseudo-random and cryptographically secure random numbers. It exposes a simple API for producing random floating-point values in [0.0, 1.0) and random integers within a caller-specified range.

## Key Functionalities

- Generate a cryptographically secure random floating-point value between 0.0 (inclusive) and 1.0 (exclusive).
- Generate a non-cryptographically-secured random integer within an inclusive-start, exclusive-end range.
- Return a typed error when the range arguments are invalid.

## Examples

```ballerina
import ballerina/io;
import ballerina/random;

public function main() returns error? {
    float d = random:createDecimal();
    io:println(d >= 0.0 && d < 1.0); // true

    int n = check random:createIntInRange(1, 100);
    io:println(n >= 1 && n < 100); // true

    int|random:Error bad = random:createIntInRange(5, 5);
    io:println(bad is random:Error); // true
}
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Cryptographically secure random float in [0.0, 1.0) | Supported | `createDecimal()` reads 53 bits from `crypto/rand` (OS entropy pool). See Notable Behavioural Changes. |
| Random integer in a caller-specified range | Supported | `createIntInRange(startRange, endRange)` uses `math/rand/v2`; range is [startRange, endRange). See Notable Behavioural Changes. |
| Error on invalid range arguments | Supported | Returns `random:Error` when `startRange >= endRange`. |
| Module-level error type | Partially Supported | `random:Error` is a plain `error` alias; `distinct` type descriptor not yet supported. |
| Arithmetic error subtype | Not Yet Supported | `random:ArithmeticError` declared as plain alias; `distinct Error` subtyping not yet supported. |

### Notable Behavioural Changes

- **`createDecimal()` — improved entropy precision.** jBallerina delegates to `java.security.SecureRandom.nextFloat()`, which returns a Java 32-bit `float` (24 bits of mantissa) widened to a 64-bit Ballerina `float`. The Go-native version reads 53 bits from `crypto/rand`, producing a full-precision IEEE 754 `float64`. The range [0.0, 1.0) is preserved; values have higher randomness quality.
- **`createIntInRange()` — corrected range distribution.** The jBallerina formula `startRange + int(rand × (endRange−1−startRange))` never produces `endRange−1` due to an off-by-one in the original implementation. The Go-native version uses `math/rand/v2.Int64N(endRange−startRange) + startRange`, which correctly produces uniform values across the full `[startRange, endRange)` range per the documented specification.
