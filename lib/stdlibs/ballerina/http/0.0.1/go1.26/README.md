# Ballerina HTTP Library

## Overview

This module provides the HTTP client and listener APIs for building and consuming HTTP services. The full jBallerina `http` module covers two sides:

**Client** — response data binding to custom types and status code records, authentication (Basic, Bearer, JWT, OAuth2), resiliency patterns (circuit breaker, retry, failover, load balancer), cookie management, HTTP response caching, compression negotiation, connection pooling, async requests, HTTP/2 server push, Server-Sent Events, multipart payloads, and streaming I/O.

**Service / Listener** — an HTTP listener with configurable host, TLS, HTTP version, and request limits; service definition with path-based routing and resource function dispatch; automatic binding of path parameters, query parameters, headers, and payloads in resource signatures; caller-based response dispatch; request/response interceptor pipeline; service-level and resource-level annotations (`@http:ServiceConfig`, `@http:ResourceConfig`, `@http:Payload`, `@http:Header`, `@http:Query`, `@http:Cache`); CORS configuration; listener authentication and authorization (File user store, LDAP, JWT, OAuth2); status code response types from resources; and SSE streaming responses.

The Go Native Interpreter currently supports the **HTTP client subset only**: the nine core remote methods (including `forward`), TLS/mTLS (PEM-based), redirect following, connection pooling, and manual payload extraction from responses. The service/listener side is not yet implemented.

## Key Functionalities

- Send HTTP requests using GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, and custom verbs.
- Forward inbound requests to upstream services preserving method, headers, and body (`forward`).
- Configure request timeout, HTTP version (1.1/2.0), redirect behaviour, connection pool settings, and compression negotiation.
- Secure connections with TLS and mutual TLS using PEM certificate and key files.
- Set custom request headers and override the inferred Content-Type.
- Read the response status code, text, JSON, or binary payload.
- Inspect response headers by name or enumerate all header names.
- Construct `Response` objects in resource functions and populate them with `setTextPayload`, `setJsonPayload`, `setBinaryPayload`, `setHeader`, and direct field assignment (`response.statusCode = 404`).
- Construct outbound `Request` objects and populate them for forwarding.
- Set a request or response body generically via `setPayload`, dispatched by the runtime type of the value.
- Decode an `application/x-www-form-urlencoded` request body into a parameter map via `getFormParams`.
- Set and extract multipart request/response bodies as `mime:Entity[]` via `setBodyParts`/`getBodyParts`.
- Parse structured header values (value + parameter map) with the header parsing utility.

## Examples

```ballerina
import ballerina/http;
import ballerina/io;

public function main() returns error? {
    // Plain HTTP client with a 10-second timeout and custom pool settings
    http:Client client = check new ("http://httpbin.org", {
        timeout: 10,
        poolConfig: {maxIdleConnections: 50}
    });

    // GET request
    http:Response getResp = check client->get("/get");
    io:println("Status: ", getResp.statusCode);
    json body = check getResp.getJsonPayload();
    io:println("Body: ", body);

    // POST request with a JSON payload
    json payload = {name: "Alice", age: 30};
    http:Response postResp = check client->post("/post", payload);
    io:println("POST status: ", postResp.statusCode);

    // Forward an inbound request to an upstream service
    http:Request req = new;
    req.method = "PUT";
    req.setJsonPayload({id: 1, data: "updated"});
    http:Response fwdResp = check client->forward("/resource/1", req);
    io:println("Forward status: ", fwdResp.statusCode);

    // TLS client with a custom CA certificate
    http:Client secureClient = check new ("https://example.com", {
        secureSocket: {cert: "/path/to/ca.pem"}
    });
    http:Response secureResp = check secureClient->get("/");
    io:println("Secure status: ", secureResp.statusCode);
}
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The tables below outline the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

### Client

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Core HTTP request methods | Supported | `get`, `post`, `put`, `patch`, `delete`, `head`, `options` are all implemented. |
| Custom HTTP verb execution | Supported | `execute` accepts any HTTP verb string. |
| Request forwarding via incoming request | Supported | `forward(path, request)` forwards the method, headers, and body of an `http:Request` to the upstream service; hop-by-hop headers are stripped per RFC 7230. |
| Request timeout | Supported | Configured via `timeout` in `ClientConfiguration` (decimal seconds, default 30). |
| HTTP version selection | Supported | `HTTP_1_0`, `HTTP_1_1`, and `HTTP_2_0` (default) are supported. Specifying `HTTP_1_0` at runtime prints a warning and falls back to HTTP/1.1, because Go's HTTP client cannot send HTTP/1.0 requests. |
| Redirect following | Supported | Full `FollowRedirects` record supported: `enabled`, `maxCount` (default 5), `allowAuthHeaders`. |
| Connection pooling | Supported | `PoolConfiguration` / `poolConfig` field in `ClientConfiguration`: `maxIdleConnections` (default 100), `maxActiveConnections` (-1 = unlimited), `waitTime` (default 30s). Defaults mirror jBallerina's pool config. |
| Custom request headers | Supported | Accepted as `map<string\|string[]>` on every method. |
| Content-Type inference from payload type | Supported | `string` → `text/plain`, `byte[]` → `application/octet-stream`, all other `json`-compatible values → `application/json`. |
| Media type override | Supported | `mediaType` parameter on body-carrying methods overrides the inferred Content-Type. |
| TLS and mutual TLS (mTLS) | Partially Supported | PEM-file-based CA trust (`cert` as a string path) and client certificate/key pairs (`key` as `CertKey`) are supported. `crypto:TrustStore`, `crypto:KeyStore`, password-protected private keys (`keyPassword`), OCSP/CRL certificate revocation (`certValidation`), and TLS session timeout (`sessionTimeout`) are not supported. |
| Client-side response data binding | Not Yet Supported | The `targetType` parameter is absent; callers must extract the payload explicitly via `getTextPayload`, `getJsonPayload`, or `getBinaryPayload`. Binding to custom record types, `xml`, or `stream<SseEvent, error?>` is not available. |
| Status code response binding | Not Yet Supported | `StatusCodeClient` and `getStatusCodeRecord()` are not implemented. |
| Client authentication | Not Yet Supported | The `auth` field in `ClientConfiguration` is absent. BasicAuth (`CredentialsConfig`), BearerToken, self-signed JWT (`JwtIssuerConfig`), and all OAuth2 grant types are not supported. |
| Circuit breaker | Not Yet Supported | `circuitBreaker` configuration and `CircuitBreakerClient` are not implemented. |
| Automatic retry | Not Yet Supported | `retryConfig` configuration and `RetryClient` are not implemented. |
| Failover client | Not Yet Supported | `FailoverClient` is not implemented. |
| Load balancer client | Not Yet Supported | `LoadBalanceClient` is not implemented. |
| Cookie management | Not Yet Supported | `cookieConfig`, `CookieStore`, and `getCookieStore()` are not implemented. |
| HTTP response caching | Not Yet Supported | The `cache` (`CacheConfig`) configuration is not implemented. |
| Compression negotiation | Supported | `COMPRESSION_AUTO` adds no `Accept-Encoding` header (server decides). `COMPRESSION_ALWAYS` adds `Accept-Encoding: deflate, gzip` if not already set. `COMPRESSION_NEVER` removes any `Accept-Encoding` header. Compressed responses (`Content-Encoding: gzip` or `deflate`) are transparently decompressed in all modes. |
| HTTP/1.x protocol settings | Not Yet Supported | `http1Settings` (keep-alive, chunking, proxy) is not implemented. |
| HTTP/2 protocol settings | Not Yet Supported | `http2Settings` (prior knowledge, initial window size) is not implemented. |
| Response size limits | Partially Supported | `responseLimits` (`ResponseLimitConfigs`) is implemented. `maxHeaderSize` maps to Go's `http.Transport.MaxResponseHeaderBytes`. `maxEntityBodySize` is enforced per-response via a counting reader (errors surface on payload extraction). `maxStatusLineLength` is accepted and validated (must be ≥ 0) but **not enforced at runtime** — Go's HTTP transport has no equivalent limit. |
| TCP socket configuration | Not Yet Supported | `socketConfig` (`ClientSocketConfig`) is not implemented. |
| Client-side payload validation | Not Yet Supported | The `validation` and `laxDataBinding` flags in `ClientConfiguration` are not implemented. |
| Proxy support | Supported | `ProxyConfig` is supported via the top-level `proxy` field in `ClientConfiguration`. Proxy auth (`userName`/`password`) is forwarded via HTTP CONNECT for HTTPS targets and `Proxy-Authorization` for HTTP targets. The deprecated `http1Settings.proxy` path is not supported (we have no `http1Settings`). DNS resolution of the proxy hostname is lazy (per-request) rather than eager at client init — initialization does not fail on an unresolvable proxy host. |
| Async request submission | Not Yet Supported | `submit`, `getResponse`, and `HttpFuture` are not implemented. |
| HTTP/2 server push | Not Yet Supported | `hasPromise`, `getNextPromise`, `getPromisedResponse`, and `rejectPromise` are not implemented. |
| Resource function call syntax | Not Yet Supported | The `client->/path.get(...)` path-template syntax is not supported; use the remote method form instead. |

### Request

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Request object construction | Supported | `new http:Request()` creates an outbound request with `rawPath`, `method`, and `httpVersion` fields. |
| Request write methods | Supported | `setTextPayload`, `setJsonPayload`, `setBinaryPayload` (each with optional `contentType`), `setHeader`, `addHeader`, `removeHeader`, `removeAllHeaders`, and `setContentType` populate the request. |
| Request read methods | Supported | `getTextPayload`, `getJsonPayload`, `getBinaryPayload`, `getHeader`, `getHeaders`, `hasHeader`, `getHeaderNames`, `getContentType`, `getQueryParams`, `getQueryParamValue`, and `getQueryParamValues` read from client-constructed or inbound requests. |
| Generic payload setter | Partially Supported | `setPayload(json\|byte[]\|mime:Entity[] payload, string? contentType = ())` dispatches by runtime type to `setTextPayload`/`setBinaryPayload`/`setJsonPayload`/`setBodyParts`. jBallerina's signature also accepts `stream<byte[], io:Error?>`/`stream<SseEvent, error?>`, not accepted here since `http` doesn't support those payload kinds yet; `anydata` values outside `json` (e.g. `xml`, `table`) are also not accepted, since there is no `toJson()`-style conversion available in this port. |
| Form-urlencoded payload parsing | Supported | `getFormParams()` decodes an `application/x-www-form-urlencoded` body into a `map<string>`. Returns an `error` if the `Content-Type` header is missing, invalid, or not `application/x-www-form-urlencoded`, or if the body cannot be decoded. |
| Multipart and form-data payload | Partially Supported | `setBodyParts(mime:Entity[] bodyParts, string? contentType = ())` / `getBodyParts() returns mime:Entity[]\|error`, built on `ballerina/mime`'s multipart support. Flat (single-level) multipart only, matching `mime`'s own limitation — a part whose own body is itself multipart is not recursively decoded. `RequestMessage` (accepted by the client's `post`/`put`/`patch`/`execute` methods) is widened to include `mime:Entity[]`. |
| Path parameter binding | Not Yet Supported | Automatic extraction of URL path segments into resource function parameters is not implemented. |
| Query parameter binding | Not Yet Supported | Automatic binding of URL query parameters to resource function parameters is not implemented. |
| Inbound header binding | Not Yet Supported | Automatic binding of request headers to resource function parameters via `@http:Header` is not implemented. |
| Inbound payload binding | Not Yet Supported | Automatic deserialization of the request body into typed resource function parameters via `@http:Payload` is not implemented. |
| Streaming request body | Not Yet Supported | `stream<byte[], io:Error?>` as a request payload type is not implemented. |

### Response

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Response status code access | Supported | Exposed as the `statusCode` field on `Response`. |
| Response payload as text | Supported | `getTextPayload()` returns the body as a `string`. |
| Response payload as JSON | Supported | `getJsonPayload()` parses the body and returns `json\|error`. |
| Response payload as raw bytes | Supported | `getBinaryPayload()` returns `byte[]\|error`. |
| Response header inspection | Supported | `hasHeader`, `getHeader`, `getHeaders`, and `getHeaderNames` operate on transport (leading) headers. Trailing header position is accepted at compile time but has no runtime effect. |
| Response object construction | Supported | `new http:Response()` creates a response with status code 200; initialised via `init()`. |
| Response write methods | Supported | `setTextPayload`, `setJsonPayload`, `setBinaryPayload` (each with optional `contentType`), `setHeader`, `addHeader`, `removeHeader`, `removeAllHeaders`, and `setContentType` populate a constructed `Response`. Status code is set by direct field assignment (`resp.statusCode = 404`). |
| Generic payload setter | Partially Supported | `setPayload(json\|byte[]\|mime:Entity[] payload, string? contentType = ())` dispatches by runtime type to `setTextPayload`/`setBinaryPayload`/`setJsonPayload`/`setBodyParts`. jBallerina's signature also accepts `stream<byte[], io:Error?>`/`stream<SseEvent, error?>`, not accepted here since `http` doesn't support those payload kinds yet; `anydata` values outside `json` (e.g. `xml`, `table`) are also not accepted, since there is no `toJson()`-style conversion available in this port. |
| Multipart and form-data payload | Partially Supported | `setBodyParts(mime:Entity[] bodyParts, string? contentType = ())` / `getBodyParts() returns mime:Entity[]\|error`, built on `ballerina/mime`'s multipart support. Flat (single-level) multipart only, matching `mime`'s own limitation. |
| Streaming response body | Not Yet Supported | `getByteStream()` is not implemented. |
| Server-Sent Events | Not Yet Supported | `getSseEventStream()` and consuming a `stream<SseEvent, error?>` response are not implemented. |
| Response XML payload | Not Yet Supported | The `xml` type and related payload handling methods (`getXmlPayload()`, `setXmlPayload()`) are not implemented due to the lack of XML support in the Go runtime; the same limitation exists in `ballerina/mime`'s `setXml`/`getXml`. |

### Listener

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| HTTP Listener | Not Yet Supported | The `Listener` class (start, graceful stop, attach, detach) is not implemented; no server-side listener can be created. |
| Listener configuration | Not Yet Supported | `ListenerConfiguration` (host, timeout, HTTP version, HTTP/1.x settings, HTTP/2 window size, graceful stop timeout, request limits, server name, socket config) is not implemented. |
| Listener TLS / mTLS | Not Yet Supported | `ListenerSecureSocket` (server certificate, mutual TLS, protocol, ciphers, etc.) is not implemented. |
| Default listener | Not Yet Supported | The module-level default listener (`http:defaultListener`) is not implemented. |
| Listener authentication and authorization | Not Yet Supported | `ListenerAuthConfig` and listener-side auth handlers (file user store, LDAP, JWT, OAuth2) are not implemented. |

### Service

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| HTTP service definition and routing | Not Yet Supported | Declaring `service on listener` with path-based routing is not implemented. |
| Resource function dispatch | Not Yet Supported | Resource functions with path parameters, accessor methods, and typed response returns are not implemented. |
| Caller-based response dispatch | Not Yet Supported | The `Caller` class and its `respond()` method for sending responses back to the client are not implemented. |
| Status code response types from resources | Not Yet Supported | Returning `http:Ok`, `http:Created`, `http:NotFound`, and other `StatusCodeResponse` subtypes from resource functions is not implemented. |
| Service-level annotation | Not Yet Supported | `@http:ServiceConfig` (host, compression, chunking, CORS, auth, validation, lax data binding) is not implemented. |
| Resource-level annotation | Not Yet Supported | `@http:ResourceConfig` (name, consumes, produces, CORS, auth, linked resources) is not implemented. |
| Response cache annotation | Not Yet Supported | `@http:Cache` on resource return types is not implemented. |
| CORS configuration | Not Yet Supported | Cross-origin resource sharing configuration at service and resource level is not implemented. |
| Request and response interceptors | Not Yet Supported | `RequestInterceptor`, `ResponseInterceptor`, `RequestErrorInterceptor`, `ResponseErrorInterceptor`, and the `InterceptableService` type are not implemented. |
| Request context | Not Yet Supported | `RequestContext` for passing data through the interceptor pipeline is not implemented. |
| Service contract type | Not Yet Supported | `ServiceContract` type for contract-first service definitions is not implemented. |
| Service-level compression and chunking | Not Yet Supported | Response compression and chunking configuration on the service side are not implemented. |
| Inbound payload validation | Not Yet Supported | Automatic constraint validation of inbound request payloads via `ballerina/constraint` is not implemented. |

### Common

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Header value parsing utility | Supported | `parseHeader()` parses comma-separated header values with parameters into `HeaderValue[]`. |
| HTTP version enum | Supported | `HttpVersion` with `HTTP_1_0`, `HTTP_1_1`, and `HTTP_2_0` enum constants. `HTTP_1_0` prints a runtime warning and falls back to HTTP/1.1. |
| Distinct HTTP error types | Not Yet Supported | All errors surface as the generic `error` type; `http:ClientError`, `http:HeaderNotFoundError`, and similar subtypes are not declared — `is http:ClientError` type checks will not work. |
| Observability and metrics | Not Yet Supported | Metrics and tracing integration via `ballerina/observe` is not implemented. |
| XML payloads | Not Yet Supported | The `xml` type and related payload handling methods (`getXmlPayload()`, `setXmlPayload()`) are not implemented due to the lack of XML support in the Go runtime; the same limitation exists in `ballerina/mime`'s `setXml`/`getXml`. |

### Notable Behavioural Changes

- **HTTP/1.0 falls back to HTTP/1.1 at runtime.** `HTTP_1_0` is present in the `HttpVersion` enum for jBallerina contract compatibility. When used, the Go runtime prints a warning to stderr and transparently upgrades the connection to HTTP/1.1, because Go's HTTP client cannot send HTTP/1.0 requests.
- **Trailing headers are not modelled.** The `TRAILING` header position constant is accepted at compile time for API compatibility, but all header operations (`getHeader`, `getHeaders`, `hasHeader`, `getHeaderNames`) act on transport (leading) headers at runtime. HTTP trailers sent by the server are silently discarded.
- **TLS protocol name has no effect.** The `protocol.name` field accepts `"SSL"`, `"TLS"`, and `"DTLS"` at compile time, but only TLS is supported at runtime. `"SSL"` and `"DTLS"` values are ignored because Go's standard TLS stack does not expose separate SSL or DTLS stacks.
- **`poolConfig.waitTime` maps to `ResponseHeaderTimeout`.** jBallerina's `waitTime` limits how long a request waits for a connection. In the Go runtime this is approximated by `ResponseHeaderTimeout` (maximum time to wait for the first response byte). True connection-wait limiting is not available in Go's `net/http` transport.
- **`responseLimits.maxStatusLineLength` is not enforced.** The value is accepted and validated (must be ≥ 0) but has no runtime effect. Go's HTTP transport does not expose a configurable maximum status line length (unlike jBallerina's Netty `HttpClientCodec`).
- **Proxy DNS resolution is lazy, not eager.** In jBallerina, `ProxyConfig.host` is DNS-resolved at client creation time, and an unknown hostname causes an `error` from `new http:Client(...)`. In the Go runtime, DNS resolution is deferred to the first request that uses the proxy. A bad proxy hostname does not fail at init time.
