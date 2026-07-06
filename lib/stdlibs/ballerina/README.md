# Ballerina Standard Library — Go Native Support

This directory contains the Go-native implementations of the `ballerina/*` standard library
packages baked into the interpreter binary. Each package is compiled into embedded `.sym`/`.bir`
artefacts and laid out as `<name>/0.0.1/go1.2/`. See each package's own README (linked below)
for the full feature-by-feature support table and behavioural notes.

## Packages

Support % is computed as `round(Supported / Total * 100)`, where *Total* is the number of rows
in each package's support table (Supported + Partially Supported + Not Yet Supported + Cannot Support).

| Package | Supported | Partially Supported | Not Yet Supported | Support % |
|---|---|---|---|---|
| [http](http/0.0.1/go1.2/README.md) | 24 | 2 | 46 | 33% |
| [io](io/0.0.1/go1.2/README.md) | 14 | 1 | 11 | 54% |
| [math.vector](math.vector/0.0.1/go1.2/README.md) | 5 | 0 | 0 | 100% |
| [time](time/0.0.1/go1.2/README.md) | 31 | 1 | 0 | 97% |
| [url](url/0.0.1/go1.2/README.md) | 3 | 0 | 1 | 75% |
| **Total** | **77** | **4** | **58** | **55%** |

## Notable Behavioural Changes

Consolidated from each package's README. Only permanent, architectural Go-level divergences are
listed here; temporary language gaps are tracked as `Not Yet Supported` rows in the per-package
tables instead.

### http

- **HTTP/1.0 is a compile error.** Specifying `httpVersion: "1.0"` (or any value outside the `HttpVersion` enum) in `ClientConfiguration` is rejected at compile time. Go's HTTP client cannot send HTTP/1.0 requests, so this is a permanent restriction rather than a missing runtime feature.
- **Trailing headers are not modelled.** The `TRAILING` header position constant is accepted at compile time for API compatibility, but all header operations (`getHeader`, `getHeaders`, `hasHeader`, `getHeaderNames`) act on transport (leading) headers at runtime. HTTP trailers sent by the server are silently discarded.
- **TLS protocol name has no effect.** The `protocol.name` field accepts `"SSL"`, `"TLS"`, and `"DTLS"` at compile time, but only TLS is supported at runtime. `"SSL"` and `"DTLS"` values are ignored because Go's standard TLS stack does not expose separate SSL or DTLS stacks.
- **`poolConfig.waitTime` maps to `ResponseHeaderTimeout`.** jBallerina's `waitTime` limits how long a request waits to acquire a connection from the pool. In the Go runtime this is approximated by `ResponseHeaderTimeout` (maximum time to wait for the first response byte). True connection-wait limiting is not available in Go's `net/http` transport.
- **`responseLimits.maxStatusLineLength` is not enforced.** The value is accepted and validated (must be ≥ 0) but has no runtime effect. Go's HTTP transport does not expose a configurable maximum status line length (unlike jBallerina's Netty `HttpClientCodec`).
- **Proxy DNS resolution is lazy, not eager.** In jBallerina, `ProxyConfig.host` is DNS-resolved at client creation time and an unknown hostname causes an `error` from `new http:Client(...)`. In the Go runtime, DNS resolution is deferred to the first request that uses the proxy. A bad proxy hostname does not fail at init time.

### io

- **`fileWriteJson` key ordering.** jBallerina writes JSON object keys in insertion order; the Go-native version writes them in **alphabetical order** — Go's `encoding/json` sorts map keys.

### time

- **`Utc` type mutability.** jBallerina declares `Utc` as `readonly & [int, decimal]` (immutable tuple). The Go-native version uses a plain mutable tuple type because `readonly &` intersection types on tuples are not yet supported by the interpreter's AST transformation. Programs should treat `Utc` values as immutable by convention; mutation is not guarded at runtime.
- **`ZoneOffset` type mutability.** Same as above — `ZoneOffset` is declared as a plain open record instead of `readonly & record {| ... |}`. Programs should not mutate `ZoneOffset` values.
- **`FormatError` is not distinct.** jBallerina's `FormatError` is a `distinct Error` subtype, allowing `error is time:FormatError` checks to distinguish it from other errors. The Go-native version declares `FormatError` as a plain `error` alias because `distinct` type descriptors are not yet supported. `error is time:FormatError` will not narrow correctly in the Go version.
- **Error message wording for `dateValidate`, `dayOfWeek`, `utcFromCivil`, `TimeZone.init`, `TimeZone.utcFromCivil`.** These functions return errors whose message text is produced by Go's standard `time` package or the Go-native implementation rather than Java's `DateTimeException.getMessage()`. The message content differs (e.g., "invalid date: 2021-02-30" vs. "Invalid value for DayOfMonth..."). Programs must not depend on the exact error message text.
- **`monotonicNow()` epoch.** The specification states the epoch is "unspecified". jBallerina uses the JVM process start (`System.nanoTime()`); the Go-native version uses the time at which the PAL was constructed. The two values are not comparable across processes and will differ between implementations. This is expected behavior.
- **Named IANA timezones in `civilToString`, `civilToEmailString`, and `TimeZone`.** When a `Civil` record carries a `timeAbbrev` containing an IANA zone name (e.g., `"Asia/Colombo"`), or when a `TimeZone` object is constructed from an IANA name, the Go-native version resolves the zone using the host operating system's timezone database via `time.LoadLocation`. If the host has an incomplete or missing IANA database, an error is returned. jBallerina ships its own bundled IANA data.
- **DST disambiguation in `TimeZone.utcFromCivil`.** When a civil time falls in an ambiguous DST window (clocks are set back), Go's `time.Date` resolves to the first (standard-time) occurrence. jBallerina honours the `which` field in the `Civil` record to select the correct occurrence. The `which` field is silently ignored in the Go-native version.

The remaining packages (`math.vector`, `url`) have **no** notable behavioural changes compared to the original jBallerina implementation for their currently supported features.
