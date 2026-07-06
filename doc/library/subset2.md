# Supported ballerina library features

Subset 2 extends the released [subset 1](subset1.md) (io console output and basic
http client) with additional `http` client configuration beyond the basic client,
the `io` file I/O surface, and the `math.vector`, `time`, and `url` modules.

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
