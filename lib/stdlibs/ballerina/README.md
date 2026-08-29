# Ballerina Standard Library — Go Native Support

This directory contains the Go-native implementations of the `ballerina/*` standard library
packages baked into the interpreter binary. Each package is compiled into embedded `.sym`/`.bir`
artefacts and laid out as `<name>/0.0.1/go1.26/`. See each package's own README (linked below)
for the full feature-by-feature support table and behavioural notes.

## Packages

Support % is computed as `round(Supported / Total * 100)`, where *Total* is the number of rows
in each package's support table (Supported + Partially Supported + Not Yet Supported + Cannot Support).

| Package                                           | Supported | Partially Supported | Not Yet Supported | Support % |
|---------------------------------------------------|---|---|---|---|
| [avro](avro/0.0.1/go1.26/README.md)               | 15 | 1 | 0 | 94% |
| [crypto](crypto/0.0.1/go1.26/README.md)           | 26 | 1 | 5 | 81% |
| [http](http/0.0.1/go1.26/README.md)               | 28 | 7 | 38 | 38% |
| [io](io/0.0.1/go1.26/README.md)                   | 21 | 2 | 4 | 78% |
| [log](log/0.0.1/go1.26/README.md)                 | 7 | 2 | 15 | 29% |
| [math.vector](math.vector/0.0.1/go1.26/README.md) | 5 | 0 | 0 | 100% |
| [os](os/0.0.1/go1.26/README.md)                   | 11 | 1 | 0 | 92% |
| [random](random/0.0.1/go1.26/README.md)           | 3 | 1 | 1 | 60% |
| [time](time/0.0.1/go1.26/README.md)               | 31 | 1 | 0 | 97% |
| [url](url/0.0.1/go1.26/README.md)                 | 3 | 0 | 1 | 75% |
| **Total**                                         | **150** | **16** | **64** | **65%** |

## Notable Behavioural Changes

Consolidated from each package's README. Only permanent, architectural Go-level divergences are
listed here; temporary language gaps are tracked as `Not Yet Supported` rows in the per-package
tables instead.

### avro

- **An unparseable schema returns an error instead of panicking.** jBallerina lets the underlying schema-parser exception escape `init`, so `new avro:Schema("not-a-schema")` panics even though `init` declares `returns Error?`; the Go-native version returns an `avro:Error` with the message `Avro schema parsing error` — this is what the module specification describes, and it stays inside the declared signature.
- **Union branch selection prefers the natural branch and applies one rule everywhere.** jBallerina uses two different rules: a union inside a record field goes through a tag table where a `float` or `double` branch also claims `int` and `decimal` values, while a top-level union bypasses that table entirely and is resolved by the underlying Avro library — which rejects `string`, `bytes`, `record`, `enum`, `array`, and `fixed` branches outright. The Go-native version applies one rule at every position: the first branch whose Avro type is the value's natural encoding wins, and only if no branch matches does the widening tag table apply. Consequently a top-level `["null","string"]` (and every other combination jBallerina rejects) works, and `["double","long"]` given an `int` selects the lossless `long` branch where a jBallerina record field would select `double`. A `bytes`/`fixed` branch and an `array` branch — or a `record` branch and a `map` branch — sharing a union are told apart by the value's own inherent type rather than by declaration order, since a Ballerina `byte[]` and a `T[]` both reach the union as the same list representation, and a record and a `map<T>` both reach it as the same mapping representation.
- **A `float` is not narrowed into an `int` or `long` schema.** jBallerina converts nothing in this case — the value falls through to Apache Avro's `GenericDatumWriter`, which casts to `java.lang.Number` and truncates toward zero, so `3.9` is stored as `3`. The Go-native version returns an `avro:Error` instead of writing a payload that does not represent the value. Conversions jBallerina performs deliberately are kept as they are: an `int` narrows into an `int` schema with the same wrapping `Long.intValue()` semantics, an `int` widens into a `float` or `double` schema, a `decimal` widens into a `double` schema, and a `string` schema stringifies whatever it is given.
- **A `bytes`/`fixed` schema requires a value whose own type is `byte[]`.** An `int[]` is rejected even when every value happens to fit in a byte (0-255) — including a bare list literal such as `schema.toAvro([1, 2, 3])`, which carries no `byte[]` type of its own since `toAvro` takes `anydata`; declare the value as `byte[]` first. jBallerina agrees this should be an error but does not implement it as one: passing a plain `int[]`, in range or not, throws an uncaught `NullPointerException` instead of returning an error.
- **Avro map keys are encoded in insertion order.** jBallerina iterates a Java `HashMap`, so the key order in an encoded Avro map is unspecified; the Go-native version writes keys in the Ballerina value's insertion order — the Avro encoding does not constrain map key order and readers are insensitive to it.
- **A `fixed` schema with `"size": 0` is rejected.** jBallerina's underlying Avro library accepts a zero-size `fixed` type, which always encodes to zero bytes. The Go-native version's underlying codec requires a size greater than zero and rejects the schema itself with an `avro:Error` from `new`. Zero-size `fixed` types have no practical use and are not expected to appear in real schemas.

### crypto

- **AES-CBC and AES-ECB always apply PKCS7 padding.** jBallerina selects PKCS5 or no padding based on the `padding` parameter value; the Go-native version always applies PKCS7 padding for CBC and ECB modes regardless of the parameter — Go's `cipher` package does not expose a separate no-padding mode. Programs relying on `NONE` padding will produce incorrect output.

### http

- **HTTP/1.0 falls back to HTTP/1.1 at runtime.** `HTTP_1_0` is present in the `HttpVersion` enum for jBallerina contract compatibility. When used, the Go runtime prints a warning to stderr and transparently upgrades the connection to HTTP/1.1, because Go's HTTP client cannot send HTTP/1.0 requests.
- **Trailing headers are not modelled.** The `TRAILING` header position constant is accepted at compile time for API compatibility, but all header operations (`getHeader`, `getHeaders`, `hasHeader`, `getHeaderNames`) act on transport (leading) headers at runtime. HTTP trailers sent by the server are silently discarded.
- **TLS protocol name has no effect.** The `protocol.name` field accepts `"SSL"`, `"TLS"`, and `"DTLS"` at compile time, but only TLS is supported at runtime. `"SSL"` and `"DTLS"` values are ignored because Go's standard TLS stack does not expose separate SSL or DTLS stacks.
- **`poolConfig.waitTime` maps to `ResponseHeaderTimeout`.** jBallerina's `waitTime` limits how long a request waits for a connection. In the Go runtime this is approximated by `ResponseHeaderTimeout` (maximum time to wait for the first response byte). True connection-wait limiting is not available in Go's `net/http` transport.
- **`responseLimits.maxStatusLineLength` is not enforced.** The value is accepted and validated (must be ≥ 0) but has no runtime effect. Go's HTTP transport does not expose a configurable maximum status line length (unlike jBallerina's Netty `HttpClientCodec`).
- **Proxy DNS resolution is lazy, not eager.** In jBallerina, `ProxyConfig.host` is DNS-resolved at client creation time, and an unknown hostname causes an `error` from `new http:Client(...)`. In the Go runtime, DNS resolution is deferred to the first request that uses the proxy. A bad proxy hostname does not fail at init time.
- **A nil target type discards the payload.** jBallerina routes a `()` target through the string payload builder, so a non-empty body is handed back as a `string` even though `()` was requested. The Go-native version returns `()` and drops the body, keeping the bound value inside the requested type.
- **Status-code error messages use the registered reason phrase.** jBallerina reports the reason phrase the server actually sent. The PAL transport contract surfaces only the status code, so the Go-native version derives the message from the status code's registered phrase (for example `Not Found` for 404). A code outside the IANA registry — 499, which nginx sends for a client-closed request, among others — has no registered phrase, and the message becomes `status code <code>` rather than being left empty.
- **A status error with an absent body keeps its reason phrase.** jBallerina extracts the error response body with the builder its `Content-Type` selects, and an extraction failure replaces the reason phrase with `http:ApplicationResponseError creation failed: <code> response payload extraction failed`. A 4xx or 5xx sent with `Content-Type: application/json` and no body at all trips that path, because a JSON decoder rejects an empty document. The Go-native version treats an absent body as having no payload, so the message stays the reason phrase and the error detail's `body` is `()`.
- **`gracefulStop` waits for in-flight requests to drain.** In jBallerina, `gracefulStop` effectively behaves like an immediate stop — it returns promptly without waiting for active requests, so calling it from within a resource on its own listener succeeds. The Go-native version implements the `http:Listener` contract literally and blocks until in-flight requests complete or the graceful-stop timeout (default 60s) elapses. A resource that calls `gracefulStop` on the listener serving it therefore self-deadlocks until the timeout elapses and then returns an error, rather than succeeding.

### io

- **`fileWriteJson` key ordering.** jBallerina writes JSON object keys in insertion order; the Go-native version writes them in **alphabetical order** — Go's `encoding/json` sorts map keys.
- **Streams are consumed via `next()`/`close()` only.** The returned streams are driven with explicit `.next()` and `.close()` calls. Iterating a stream with a `foreach` statement or a query (`from ... in`) expression is not yet supported at the language level, so those constructs cannot yet consume these streams.
- **Write-from-stream accepts a generic `error?` completion.** jBallerina declares `fileWriteLinesFromStream`/`fileWriteBlocksFromStream` with a `stream<_, io:Error?>` parameter, which rejects a stream held as `stream<_, error?>` (e.g. `stream<byte[], error?> s = check io:fileReadBlocksAsStream(p); check io:fileWriteBlocksFromStream(out, s);` fails to compile in jBallerina). This port widens the parameter completion type to the generic `error?`, so both `io:Error?` and plain `error?` completion streams are accepted. This is a strict superset — every jBallerina-valid call still compiles — and the return type remains the specific `io:Error?`.
- **`writeVarInt`/`readVarInt` round-trip the full `int` range.** jBallerina's variable-length integer implementation breaks for very large magnitudes: `readVarInt` panics on encodings longer than 8 bytes (so its own `writeVarInt` output for values needing 9-10 bytes cannot be read back), and `writeVarInt(int:MIN_VALUE)` silently writes a single `0x00` byte. This port encodes such values with the minimal correct width and reads encodings up to 10 bytes, so every `int` round-trips; the wire format is identical to jBallerina for all values jBallerina handles correctly.
- **Unknown charset handling differs between `writeString` and `readString`.** `WritableDataChannel.writeString` surfaces an unsupported charset as a Go error, which the interpreter turns into a panic, matching jBallerina's unchecked `UnsupportedEncodingException`. `ReadableDataChannel.readString` instead returns it as an `io:Error` value.

### log

- **Module name always empty.** jBallerina uses JVM `StackWalker` to detect the calling module name at runtime; the Go-native version has no equivalent mechanism, so `module=""` in all log records.
- **Error field format.** jBallerina serialises a full `FullErrorDetails` record (message, stack trace, cause chain) for the `error` field; the Go-native version formats the error as `error("message")` using the Ballerina `toBalString` representation of the error value.

### os

- **Environment mutations are process-wide.** jBallerina uses per-strand env maps for isolation; the Go-native version calls `os.Setenv` / `os.Unsetenv` directly, mutating the process-wide environment. This is safe for single-threaded Ballerina programs but not for concurrent strand execution.

### random

- **`createDecimal()` — improved entropy precision.** jBallerina delegates to `java.security.SecureRandom.nextFloat()`, which returns a Java 32-bit `float` (24 bits of mantissa) widened to a 64-bit Ballerina `float`. The Go-native version reads 53 bits from `crypto/rand`, producing a full-precision IEEE 754 `float64`. The range [0.0, 1.0) is preserved; values have higher randomness quality.
- **`createIntInRange()` — corrected range distribution.** The jBallerina formula `startRange + int(rand × (endRange−1−startRange))` never produces `endRange−1` due to an off-by-one in the original implementation. The Go-native version uses `math/rand/v2.Int64N(endRange−startRange) + startRange`, which correctly produces uniform values across the full `[startRange, endRange)` range per the documented specification.

### time

- **`Utc` type mutability.** jBallerina declares `Utc` as `readonly & [int, decimal]` (immutable tuple). The Go-native version uses a plain mutable tuple type because `readonly &` intersection types on tuples are not yet supported by the interpreter's AST transformation. Programs should treat `Utc` values as immutable by convention; mutation is not guarded at runtime.
- **`ZoneOffset` type mutability.** Same as above — `ZoneOffset` is declared as a plain open record instead of `readonly & record {| ... |}`. Programs should not mutate `ZoneOffset` values.
- **`FormatError` is not distinct.** jBallerina's `FormatError` is a `distinct Error` subtype, allowing `error is time:FormatError` checks to distinguish it from other errors. The Go-native version declares `FormatError` as a plain `error` alias because `distinct` type descriptors are not yet supported. `error is time:FormatError` will not narrow correctly in the Go version.
- **Error message wording for `dateValidate`, `dayOfWeek`, `utcFromCivil`, `TimeZone.init`, `TimeZone.utcFromCivil`.** These functions return errors whose message text is produced by Go's standard `time` package or the Go-native implementation rather than Java's `DateTimeException.getMessage()`. The message content differs (e.g., "invalid date: 2021-02-30" vs. "Invalid value for DayOfMonth..."). Programs must not depend on the exact error message text.
- **`monotonicNow()` epoch.** The specification states the epoch is "unspecified". jBallerina uses the JVM process start (`System.nanoTime()`); the Go-native version uses the time at which the PAL was constructed. The two values are not comparable across processes and will differ between implementations. This is expected behavior.
- **Named IANA timezones in `civilToString`, `civilToEmailString`, and `TimeZone`.** When a `Civil` record carries a `timeAbbrev` containing an IANA zone name (e.g., `"Asia/Colombo"`), or when a `TimeZone` object is constructed from an IANA name, the Go-native version resolves the zone using the host operating system's timezone database via `time.LoadLocation`. If the host has an incomplete or missing IANA database, an error is returned. jBallerina ships its own bundled IANA data.
- **DST disambiguation in `TimeZone.utcFromCivil`.** When a civil time falls in an ambiguous DST window (clocks are set back), Go's `time.Date` resolves to the first (standard-time) occurrence. jBallerina honours the `which` field in the `Civil` record to select the correct occurrence. The `which` field is silently ignored in the Go-native version.

The remaining packages (`math.vector`, `url`) have **no** notable behavioural changes compared to the original jBallerina implementation for their currently supported features.
