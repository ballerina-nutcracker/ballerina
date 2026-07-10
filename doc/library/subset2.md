# Supported ballerina library features

Subset 2 extends the released [subset 1](subset1.md) (io console output and basic
http client) with the core surface of the `crypto`, `io` (file I/O), `log`, `os`,
`random`, `math.vector`, `time`, and `url` modules, plus an `http` server/listener
and additional `http` client configuration beyond the basic client.

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

### Server (listener & service)

Subset 2 adds the server side: an `http:Listener` that accepts connections and
routes requests to attached services by base path.

| Feature | Notes |
|---|---|
| `new http:Listener(port, config?)` | `ListenerConfiguration`: `host` (default `0.0.0.0`), `timeout` (seconds, default 60), `httpVersion` (`http:HTTP_2_0` default — serves HTTP/1.1 + HTTP/2 — or `http:HTTP_1_1`), `secureSocket` |
| `secureSocket` (TLS) | `ListenerSecureSocket`: `key` (`CertKey` with `certFile`/`keyFile`), `cert`, `mutualSsl`, `protocol`, `ciphers`, `shareSession` |
| `service /basePath on new http:Listener(port) { ... }` | Attach a service at a base path declaratively, or via `Listener.attach(svc, name?)` / `detach(svc)` |
| Resource functions | `resource function get\|post\|... <path>(...)`; path segments may be typed params (`[int id]`, `[string s]`, `[boolean b]`, `[decimal d]`) coerced from the URL; an optional `http:Request` parameter receives the inbound request |
| Resource return | `http:Response`, `error`, `()`, or unions thereof — `()` → 202, `error` → 500, `http:Response` is written as-is. Bare `string`/scalar/`anydata` returns are not supported in this cut (construct an `http:Response`). Non-matching path → 404, wrong method → 405 |
| Listener lifecycle | `start` / `gracefulStop` / `immediateStop` are driven by the module lifecycle (`$start`/`$gracefulStop`/`$immediateStop`); the program stays alive while listening and winds down on a stop signal |

### Request / Response messages

Subset 1 exposed the response payload/header getters. Subset 2 adds mutating
methods (used when building responses in services and forwarding requests):

| Method | Notes |
|---|---|
| `addHeader` / `removeHeader` / `removeAllHeaders` | Add/remove headers without replacing existing values; `position` (`LEADING`/`TRAILING`) accepted, `TRAILING` ignored at runtime |
| `setContentType` / `getContentType` | Set/read the `Content-Type` header |

`http:Response` also exposes the `statusCode` (default 200), `reasonPhrase`,
`server`, and `resolvedRequestedURI` fields. The `forward` client method proxies
a received `http:Request` to a target unchanged.

Subset 2 also adds request payload setters and query parameter access, used
when constructing or forwarding requests:

| Method | Notes |
|---|---|
| `setTextPayload` / `setJsonPayload` / `setBinaryPayload` | Populate a request or response body, each with an optional `contentType` override |
| `getQueryParams` / `getQueryParamValue` / `getQueryParamValues` | Read URL query parameters from an `http:Request`; `()` when absent or when the request has no parsed query string (e.g. a client-constructed request) |

## [crypto](https://github.com/ballerina-platform/module-ballerina-crypto/blob/master/docs/spec/spec.md)

### Hashing, HMAC, and checksums

| Function | Notes |
|---|---|
| `hashMd5` / `hashSha1` / `hashSha256` / `hashSha384` / `hashSha512` | Digest of a `byte[]`, with an optional salt prepended before hashing |
| `hashKeccak256` | Legacy (pre-standardisation) Keccak-256 digest, distinct from SHA3-256 |
| `crc32b` | CRC32B checksum; returns an 8-character uppercase hex string |
| `hmacMd5` / `hmacSha1` / `hmacSha256` / `hmacSha384` / `hmacSha512` | HMAC of a `byte[]` under a `byte[]` key; returns `byte[]\|crypto:Error` |
| `equalConstantTime` | Constant-time comparison of two `HashValue` (`byte[]\|string`) values |

### Password hashing

| Function | Notes |
|---|---|
| `hashBcrypt` / `verifyBcrypt` | BCrypt hash and verify; `workFactor` configurable |
| `hashArgon2` / `verifyArgon2` | Argon2id hash and verify; iterations, memory, and parallelism configurable |
| `hashPbkdf2` / `verifyPbkdf2` | PBKDF2 hash and verify; `crypto:SHA1`, `crypto:SHA256` (default), or `crypto:SHA512`, with a configurable iteration count |

### AES encryption

| Function | Notes |
|---|---|
| `encryptAesCbc` / `decryptAesCbc` | AES-CBC; PKCS7 padding is always applied regardless of the `padding` argument |
| `encryptAesEcb` / `decryptAesEcb` | AES-ECB; PKCS7 padding is always applied regardless of the `padding` argument |
| `encryptAesGcm` / `decryptAesGcm` | AES-GCM with a configurable tag size in bits (default 128) |

### RSA and ECDSA

| Function | Notes |
|---|---|
| `encryptRsaEcb` / `decryptRsaEcb` | RSA encrypt/decrypt; accepts PKCS1 (default) or OAEP padding (`OAEPwithMD5andMGF1`, `OAEPWithSHA1AndMGF1`, `OAEPWithSHA256AndMGF1`, `OAEPwithSHA384andMGF1`, `OAEPwithSHA512andMGF1`) |
| `signRsaMd5` / `signRsaSha1` / `signRsaSha256` / `signRsaSha384` / `signRsaSha512` | RSA PKCS1v15 signing |
| `verifyRsaMd5Signature` / `verifyRsaSha1Signature` / `verifyRsaSha256Signature` / `verifyRsaSha384Signature` / `verifyRsaSha512Signature` | RSA PKCS1v15 signature verification |
| `signRsaSsaPss256` / `verifyRsaSsaPss256Signature` | RSA-PSS signing and verification (SHA-256, salt length = hash length) |
| `signSha256withEcdsa` / `signSha384withEcdsa` | ECDSA signing; DER-encoded ASN.1 signatures |
| `verifySha256withEcdsaSignature` / `verifySha384withEcdsaSignature` | ECDSA signature verification |

Signing, verifying, encrypting, or decrypting with a key of the wrong algorithm
(e.g. an EC key passed to an RSA function) returns a `crypto:Error`.

### Key loading and key derivation

| Function | Notes |
|---|---|
| `decodeRsaPrivateKeyFromKeyFile` / `decodeEcPrivateKeyFromKeyFile` | Private key from a PEM file (PKCS8, PKCS1, EC); encrypted keys (PBE, PBES2, 3DES, RC2) supported via an optional password |
| `decodeRsaPrivateKeyFromContent` | RSA private key from PEM bytes, with an optional password for encrypted keys |
| `decodeRsaPublicKeyFromCertFile` / `decodeEcPublicKeyFromCertFile` | Public key from a PEM or DER X.509 certificate file |
| `decodeRsaPublicKeyFromContent` | RSA public key from PEM bytes |
| `buildRsaPublicKey` | Construct an RSA public key from a hex-encoded modulus and exponent |
| `hkdfSha256` | HKDF-SHA256 key derivation with optional salt and info |

`crypto:Error` is a plain `error` alias; the `distinct` error subtypes are not
yet supported. RSA/EC key loading from a PKCS12 keystore or truststore
(`decode*FromKeyStore`, `decode*FromTrustStore`) is implemented but not yet part
of this subset's tested surface.

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
| `exec` | Spawn a subprocess, merging the parent environment with any overrides passed via `envProperties`; returns `os:Error` for a non-existent command |
| `Process.waitForExit` | Wait for the subprocess to exit; returns the exit code |
| `Process.output` | Read the subprocess's captured stdout (default) or stderr, after it exits |
| `Process.exit` | Terminate the subprocess immediately |

`os:Error` and `os:ProcessExecError` are plain `error` aliases; the `distinct`
error subtypes are not yet supported.

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
| `civilAddDuration` | Add a calendar `Duration` (years/months/days/hours/minutes/seconds; `weeks` normalised to days) to a `Civil` value, timezone-agnostic |
| `dateValidate` / `dayOfWeek` | Validate a `Date`; day-of-week of a `Date` |
| `getZone` | Load a named IANA timezone (`nil` for an invalid zone ID) |

### Timezone-aware operations (`TimeZone` class)

| Feature | Notes |
|---|---|
| `new time:TimeZone(id)` | Construct from an IANA name (e.g. `"Asia/Colombo"`, resolved via the host's timezone database), a fixed offset string (e.g. `"+05:30"`), or `"UTC"` |
| `TimeZone.utcToCivil` / `TimeZone.utcFromCivil` | Convert between `Utc` and `Civil` within the timezone |
| `TimeZone.civilAddDuration` | Add a `Duration` to a `Civil` value within the timezone |
| `TimeZone.fixedOffset` | Return the zone's `ZoneOffset` if it always has a constant UTC offset, or `()` otherwise |

## [url](https://github.com/ballerina-platform/module-ballerina-url/blob/master/docs/spec/spec.md)

| Function | Notes |
|---|---|
| `encode(url, charset)` | Percent-encode a URL or URL part |
| `decode(url, charset)` | Decode a percent-encoded URL or URL part |

Character encodings supported: UTF-8, ISO-8859-1, US-ASCII, UTF-16, UTF-16BE,
UTF-16LE.
