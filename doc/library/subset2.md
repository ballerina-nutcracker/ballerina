# Supported ballerina library features

Subset 2 extends the released [subset 1](subset1.md) (io console output and basic
http client) with the core surface of the `crypto`, `io` (file I/O), `log`, `os`,
`random`, `math.vector`, `time`, and `url` modules, plus additional `http` client
configuration beyond the basic client.

## [http](https://github.com/ballerina-platform/module-ballerina-http/blob/master/docs/spec/spec.md)

Subset 1 covers the basic http client (initialisation, remote methods, request
body, response payload and headers, and TLS). Subset 2 adds the following
`ClientConfiguration` fields and a header-parsing utility.

| Feature | Notes |
|---|---|
| `compression` | `http:COMPRESSION_AUTO` (default), `http:COMPRESSION_ALWAYS`, `http:COMPRESSION_NEVER` control request `Accept-Encoding` / response decompression |
| `proxy` | `ProxyConfig` (host, port, and optional credentials) routes client requests through an HTTP proxy |
| `responseLimits` | `ResponseLimitConfigs`: `maxStatusLineLength`, `maxHeaderSize`, `maxEntityBodySize` bound the accepted response size |
| `parseHeader(headerValue)` | Module-level function that parses a header value into its base value and parameter map; returns `[string, map<string>]\|http:ClientError` |

## [crypto](https://github.com/ballerina-platform/module-ballerina-crypto/blob/master/docs/spec/spec.md)

| Function | Notes |
|---|---|
| `hashSha256` | SHA-256 digest of a `byte[]`, with optional salt prepend |
| `hmacSha256` | HMAC-SHA256 of a `byte[]` under a `byte[]` key; returns `byte[]\|crypto:Error` |

## [io](https://github.com/ballerina-platform/module-ballerina-io/blob/master/docs/spec/spec.md)
### File I/O

Subset 1 covers console I/O (`print`, `println`). Subset 2 adds whole-file read
and write operations. All write functions accept an optional
`io:FileWriteOption` (`io:OVERWRITE`, the default, or `io:APPEND`).

| Function | Notes |
|---|---|
| `fileReadString` / `fileWriteString` | Read/write a file as a `string`. Line endings normalised to `\n`; trailing newline stripped on read |
| `fileReadLines` / `fileWriteLines` | Read/write a file as a `string[]`. `\n` appended after each line on write |
| `fileReadBytes` / `fileWriteBytes` | Read/write a file as a `byte[]` |
| `fileReadJson` / `fileWriteJson` | Read/write a file as `json`. `fileWriteJson` always overwrites; object keys are sorted alphabetically |
| `fileReadXml` / `fileWriteXml` | Read/write a file as `xml` |

`io:Error` is the module-level error type returned by the file operations. It is
a plain `error` alias; the `distinct` error subtypes are not yet supported.

## [log](https://github.com/ballerina-platform/module-ballerina-log/blob/master/docs/spec/spec.md)

| Function | Notes |
|---|---|
| `printDebug` | Emit a `DEBUG`-level message |
| `printInfo` | Emit an `INFO`-level message |
| `printWarn` | Emit a `WARN`-level message |
| `printError` | Emit an `ERROR`-level message |

- Each print function accepts an optional `error` value via the `'error` named
  parameter and arbitrary key-value annotations via rest-record syntax
  (e.g. `id = 845315, path = "/api"`). Key-value values are restricted to
  `anydata`.
- The log level is fixed at `INFO`: `DEBUG` messages are silently suppressed;
  `INFO`, `WARN`, and `ERROR` messages are emitted.
- Output is written to stderr in LOGFMT format:
  `time=<RFC3339> level=<LEVEL> module="" message="<msg>" [error=<err>] [key=value ...]`.

## [os](https://github.com/ballerina-platform/module-ballerina-os/blob/master/docs/spec/spec.md)

| Function | Notes |
|---|---|
| `getEnv` | Read an environment variable; returns the empty string when unset |
| `setEnv` | Set an environment variable; validates the key is not empty or `"=="`; returns `os:Error?` |
| `unsetEnv` | Unset an environment variable; validates the key is not empty; returns `os:Error?` |
| `listEnv` | Return a `map<string>` snapshot of all environment variables |
| `getUsername` | Return the current user's name |
| `getUserHome` | Return the current user's home directory |

Subprocess execution (`exec` and the `Process` object) is supported by the
module but is not exercised by subset 2 corpus tests.

## [random](https://github.com/ballerina-platform/module-ballerina-random/blob/master/docs/spec/spec.md)

| Function | Notes |
|---|---|
| `createDecimal` | Cryptographically secure random `float` in `[0.0, 1.0)` |
| `createIntInRange` | Random `int` in `[startRange, endRange)`; returns `random:Error` when `startRange >= endRange` |

`random:Error` is a plain `error` alias; the `distinct` type descriptor is not
yet supported.

## [math.vector](https://github.com/ballerina-platform/module-ballerina-math.vector/blob/master/docs/spec/spec.md)

Vector math operations over `float[]` vectors.

| Function | Notes |
|---|---|
| `vectorNorm(v, norm)` | L1 or L2 norm, selected by the `vector:L1` / `vector:L2` enum |
| `dotProduct(v1, v2)` | Dot product; panics if the vectors differ in length |
| `cosineSimilarity(v1, v2)` | Cosine similarity; panics on a zero vector |
| `euclideanDistance(v1, v2)` | Euclidean distance; panics if the vectors differ in length |
| `manhattanDistance(v1, v2)` | Manhattan distance; panics if the vectors differ in length |

## [time](https://github.com/ballerina-platform/module-ballerina-time/blob/master/docs/spec/spec.md)

UTC and civil (local) time, time zones, RFC 3339 / RFC 5322 formatting, and
duration-based date arithmetic.

Types: `Utc`, `Civil`, `Date`, `TimeOfDay`, `Seconds`, `ZoneOffset`, `Zone`,
`TimeZone`, `Duration`, `DayOfWeek`, and the related enums/constants
(`HeaderZoneHandling`, `UtcZoneHandling`, `Z`).

| Function | Notes |
|---|---|
| `utcNow` | Current UTC time (via the platform clock) |
| `utcFromString` / `utcToString` | Parse/format RFC 3339 timestamps |
| `utcAddSeconds` / `utcDiffSeconds` | Add seconds to / difference between `Utc` values |
| `utcToCivil` / `utcFromCivil` | Convert between `Utc` and `Civil` |
| `civilFromString` / `civilToString` | Parse/format RFC 3339 civil strings (incl. RFC 9557 IANA zone annotation) |
| `civilFromEmailString` / `civilToEmailString` / `utcToEmailString` | Parse/format RFC 5322 (email) date strings |
| `dateValidate` / `dayOfWeek` | Validate a `Date`; day-of-week of a `Date` |
| `getZone` | Load a named IANA timezone (`nil` for an invalid zone ID) |

## [url](https://github.com/ballerina-platform/module-ballerina-url/blob/master/docs/spec/spec.md)

| Function | Notes |
|---|---|
| `encode(url, charset)` | Percent-encode a URL or URL part |
| `decode(url, charset)` | Decode a percent-encoded URL or URL part |

Character encodings supported: UTF-8, ISO-8859-1, US-ASCII, UTF-16, UTF-16BE,
UTF-16LE.
