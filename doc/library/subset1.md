# Supported ballerina library features

## [io](https://github.com/ballerina-platform/module-ballerina-io/blob/master/docs/spec/spec.md)
### [Console I/O](https://github.com/ballerina-platform/module-ballerina-io/blob/master/docs/spec/spec.md#2-console-io)
- `print`
- `println`

## [http](https://github.com/ballerina-platform/module-ballerina-http/blob/master/docs/spec/spec.md)
### Client — remote methods

All eight remote methods are supported: `get`, `post`, `put`, `patch`, `delete`, `head`, `options`, and `execute`. Each method accepts optional request headers as a `map<string|string[]>`. Methods that carry a body (`post`, `put`, `patch`, `delete`, `execute`) additionally accept an optional media type override.

### Client — initialisation

The client can be initialised with a URL and an optional `ClientConfiguration` record. The configuration supports:

| Field | Notes |
|---|---|
| `timeout` | Request timeout as a decimal (seconds); default is `30` |
| `httpVersion` | `"1.1"` or `"2.0"` (default). HTTP/2 is enabled over both TLS (via ALPN) and cleartext (h2c) |
| `followRedirects` | Full `FollowRedirects` record: `enabled`, `maxCount` (default 5), `allowAuthHeaders` |
| `secureSocket` | See TLS section below |

### Request message body

The `message` parameter is typed as `json`, which in Ballerina includes `string`, `byte[]`, and all JSON-compatible values. The runtime infers `Content-Type` from the value:

- `string` — sent as `text/plain`
- `byte[]` (a list where every element is an integer in 0–255) — sent as `application/octet-stream`
- All other `json`-compatible values (`nil`, `boolean`, `int`, `float`, `decimal`, nested maps and lists, JSON arrays) — serialised and sent as `application/json`

The `mediaType` parameter overrides the inferred `Content-Type` in all cases.

### Response — fields

| Field | Notes |
|---|---|
| `statusCode` | HTTP status code of the response |

### Response — methods

| Method | Notes |
|---|---|
| `getTextPayload()` | Returns the response body as a string |
| `getJsonPayload()` | Parses the body as JSON; returns `json\|error` |
| `getBinaryPayload()` | Returns the body as a byte array; returns `byte[]\|error` |
| `hasHeader(name, position?)` | Returns `true` if the header is present |
| `getHeader(name, position?)` | Returns the first value for the header, or an error if absent |
| `getHeaders(name, position?)` | Returns all values for the header, or an error if absent |
| `getHeaderNames(position?)` | Returns the names of all response headers |

The `position` parameter accepts `http:LEADING` (default) or `http:TRAILING`. Trailing headers are accepted at compile time but not modelled at runtime — all operations act on transport headers.

### TLS (`secureSocket`)

| Setting | Notes |
|---|---|
| `enable` / `verifyHostName` | Disabling either turns off certificate/hostname verification |
| `cert` (string path) | Custom CA trust store from a PEM file; CN-based hostname fallback supported for legacy self-signed certificates |
| `key` (`CertKey`) | Mutual TLS using `certFile` and `keyFile` (unencrypted PEM) |
| `serverName` | Overrides the SNI hostname sent during the TLS handshake |
| `ciphers` | IANA cipher suite names applied to TLS 1.2 connections; unknown names are silently skipped |
| `handshakeTimeout` | Maximum duration allowed for the TLS handshake |
| `shareSession = false` | Disables TLS session ticket reuse |
| `protocol.versions` | Accepts `"TLSv1.0"`, `"TLSv1.1"`, `"TLSv1.2"`, `"TLSv1.3"` to set minimum and maximum TLS versions |

