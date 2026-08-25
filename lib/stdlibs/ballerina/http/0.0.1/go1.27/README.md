# Ballerina HTTP Library

## Overview

This module provides the HTTP client and listener APIs for building and consuming HTTP services. The full jBallerina `http` module covers two sides:

**Client** — response data binding to custom types and status code records, authentication (Basic, Bearer, JWT, OAuth2), resiliency patterns (circuit breaker, retry, failover, load balancer), cookie management, HTTP response caching, compression negotiation, connection pooling, async requests, HTTP/2 server push, Server-Sent Events, multipart payloads, and streaming I/O.

**Service / Listener** — an HTTP listener with configurable host, TLS, HTTP version, and request limits; service definition with path-based routing and resource function dispatch; automatic binding of path parameters, query parameters, headers, and payloads in resource signatures; caller-based response dispatch; request/response interceptor pipeline; service-level and resource-level annotations (`@http:ServiceConfig`, `@http:ResourceConfig`, `@http:Payload`, `@http:Header`, `@http:Query`, `@http:Cache`); CORS configuration; listener authentication and authorization (File user store, LDAP, JWT, OAuth2); status code response types from resources; and SSE streaming responses.

The Go Native Interpreter supports the **HTTP client subset**: the nine core remote methods (including `forward`), TLS/mTLS (PEM-based), redirect following, connection pooling, and both manual payload extraction and automatic response data binding to the contextually expected type. It also supports a native **listener**: creating an `http:Listener`, attaching and detaching services, and starting and stopping it (`'start`, `gracefulStop`, `immediateStop`); attached services are dispatched by path-based routing to resource functions. See the Listener and Service tables below for the current support status of listener configuration, TLS, and broader service-side features.

## Key Functionalities

- Send HTTP requests using GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS, and custom verbs.
- Forward inbound requests to upstream services preserving method, headers, and body (`forward`).
- Configure request timeout, HTTP version (1.1/2.0), redirect behaviour, connection pool settings, and compression negotiation.
- Secure connections with TLS and mutual TLS using PEM certificate and key files.
- Set custom request headers and override the inferred Content-Type.
- Read the response status code, text, JSON, or binary payload.
- Bind the response payload directly to the expected type — records, arrays, maps, `json`, `string`, `byte[]`, and their nilable forms — driven by the response `Content-Type`.
- Inspect response headers by name or enumerate all header names.
- Construct `Response` objects in resource functions and populate them with `setTextPayload`, `setJsonPayload`, `setBinaryPayload`, `setHeader`, and direct field assignment (`response.statusCode = 404`).
- Construct outbound `Request` objects and populate them for forwarding.
- Parse structured header values (value + parameter map) with the header parsing utility.

## Examples

```ballerina
import ballerina/http;
import ballerina/io;

public function main() returns error? {
    // Plain HTTP client with a 10-second timeout and custom pool settings
    http:Client httpClient = check new ("http://httpbin.org", {
        timeout: 10,
        poolConfig: {maxIdleConnections: 50}
    });

    // GET request
    http:Response getResp = check httpClient->get("/get");
    io:println("Status: ", getResp.statusCode);
    json body = check getResp.getJsonPayload();
    io:println("Body: ", body);

    // POST request with a JSON payload
    json payload = {name: "Alice", age: 30};
    http:Response postResp = check httpClient->post("/post", payload);
    io:println("POST status: ", postResp.statusCode);

    // Forward an inbound request to an upstream service
    http:Request req = new;
    req.method = "PUT";
    req.setJsonPayload({id: 1, data: "updated"});
    http:Response fwdResp = check httpClient->forward("/resource/1", req);
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
| Content-Type inference from payload type | Supported | `string` → `text/plain`, `xml` → `application/xml`, `byte[]` → `application/octet-stream`, all other `anydata` values → `application/json`. `RequestMessage` is `anydata\|Request`; `mime:Entity[]` and `stream<byte[], io:Error?>` messages are not supported. |
| Media type override | Supported | `mediaType` parameter on body-carrying methods overrides the inferred Content-Type. |
| TLS and mutual TLS (mTLS) | Partially Supported | PEM-file-based CA trust (`cert` as a string path) and client certificate/key pairs (`key` as `CertKey`) are supported. `crypto:TrustStore`, `crypto:KeyStore`, password-protected private keys (`keyPassword`), OCSP/CRL certificate revocation (`certValidation`), and TLS session timeout (`sessionTimeout`) are not supported. |
| Client-side response data binding | Partially Supported | Every method except `head` takes a `TargetType targetType = <>` parameter inferred from the contextually expected type. An `http:Response` target (or any union containing it) yields the raw response; any other `anydata` target is deserialised from the body according to the `Content-Type` header. `stream<SseEvent, error?>` targets are not available, and jBallerina's `data.jsondata` projection options are not applied. An `application/xml` or `text/xml` body binds to an `xml` target, including a narrower one such as `xml:Element` when the parsed value inhabits it; an empty body for a non-nilable `xml` target returns an error named `NoContentError`, unlike the `string` and `byte[]` builders below. Conversion uses the same routine as `lang.value:fromJsonWithType`, so declared union member order is not honoured and record field defaults are not injected. An empty body binds to `()` for a nilable target; for a non-nilable `string` or `byte[]` target it binds to `""` or `[]` rather than raising jBallerina's `NoContentError`, which is not declared here. A target narrower than the type its builder produces — an enum or a singleton for `text/plain`, a closed all-string record for `application/x-www-form-urlencoded`, a tuple or fixed-length array for `application/octet-stream` — is converted to that target, and a body that does not fit it returns an `error`. The nilable form of such a target (`Colour?`, `Form?`) binds the same way, with an absent body giving `()`; a narrow target that is not nilable instead receives the builder's empty value and rejects it unless the narrow type admits it. |
| Status code response binding | Not Yet Supported | `StatusCodeClient` and `getStatusCodeRecord()` are not implemented. |
| Status code error mapping | Partially Supported | When data binding is active, a 4xx or 5xx response yields an `error` whose message is the reason phrase and whose detail carries `statusCode`, `headers`, and `body`. The distinct types `http:ClientRequestError` and `http:RemoteServerError` are not declared, so the two cases cannot be distinguished with an `is` check, and `error:detail()` is not available to read the detail. |
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
| Request write methods | Supported | `setTextPayload`, `setJsonPayload`, `setXmlPayload`, `setBinaryPayload` (each with optional `contentType`), `setHeader`, `addHeader`, `removeHeader`, `removeAllHeaders`, and `setContentType` populate the request. All four payload setters follow jBallerina's rule of keeping an existing `Content-Type` when no override is passed, falling back to their type-specific default only when none is set. |
| Request read methods | Supported | `getTextPayload`, `getJsonPayload`, `getXmlPayload`, `getBinaryPayload`, `getHeader`, `getHeaders`, `hasHeader`, `getHeaderNames`, `getContentType`, `getQueryParams`, `getQueryParamValue`, and `getQueryParamValues` read from client-constructed or inbound requests. |
| Path parameter binding | Not Yet Supported | Automatic extraction of URL path segments into resource function parameters is not implemented. |
| Query parameter binding | Not Yet Supported | Automatic binding of URL query parameters to resource function parameters is not implemented. |
| Inbound header binding | Not Yet Supported | Automatic binding of request headers to resource function parameters via `@http:Header` is not implemented. |
| Inbound payload binding | Not Yet Supported | Automatic deserialization of the request body into typed resource function parameters via `@http:Payload` is not implemented. |
| Multipart and form-data payload | Not Yet Supported | `mime:Entity[]` as a request body type and the associated `getBodyParts()` response method are not implemented. |
| Streaming request body | Not Yet Supported | `stream<byte[], io:Error?>` as a request payload type is not implemented. |

### Response

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Response status code access | Supported | Exposed as the `statusCode` field on `Response`. |
| Response payload as text | Supported | `getTextPayload()` returns `string\|error`. |
| Response payload as JSON | Supported | `getJsonPayload()` parses the body and returns `json\|error`. |
| Response payload as XML | Supported | `getXmlPayload()` parses the body and returns `xml\|error`. An empty body returns an error named `NoContentError` with the message `No content`, since an empty body has no valid `xml` representation. |
| Response payload as raw bytes | Supported | `getBinaryPayload()` returns `byte[]\|error`. |
| Response header inspection | Supported | `hasHeader`, `getHeader`, `getHeaders`, and `getHeaderNames` operate on transport (leading) headers. Trailing header position is accepted at compile time but has no runtime effect. |
| Response object construction | Supported | `new http:Response()` creates a response with status code 200; initialised via `init()`. |
| Response write methods | Supported | `setTextPayload`, `setJsonPayload`, `setXmlPayload`, `setBinaryPayload` (each with optional `contentType`), `setHeader`, `addHeader`, `removeHeader`, `removeAllHeaders`, and `setContentType` populate a constructed `Response`. All four payload setters follow jBallerina's rule of keeping an existing `Content-Type` when no override is passed, falling back to their type-specific default only when none is set. Status code is set by direct field assignment (`resp.statusCode = 404`). |
| Streaming response body | Not Yet Supported | `getByteStream()` is not implemented. |
| Server-Sent Events | Not Yet Supported | `getSseEventStream()` and consuming a `stream<SseEvent, error?>` response are not implemented. |

### Listener

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| HTTP Listener | Supported | The `Listener` class is implemented: `init`, `attach`, `detach`, `'start`, `gracefulStop`, and `immediateStop` create a listener, attach/detach services, and start/stop the server. See the `gracefulStop` behavioural note below. |
| Listener configuration | Partially Supported | `ListenerConfiguration` supports `host` (default `0.0.0.0`), `timeout` (response write timeout, default 60s), `httpVersion` (`HTTP_1_1` or `HTTP_2_0`), and `secureSocket` (TLS). Request/response size limits, server name, `http1Settings`, `http2Settings`, and `socketConfig` are not present in the record. |
| Listener TLS / mTLS | Partially Supported | `ListenerSecureSocket` supports a PEM server certificate/key (`key`), mutual TLS (`mutualSsl` + `cert` as the CA path), TLS protocol version bounds (`protocol`), cipher suite selection (`ciphers`), and session ticket disabling (`shareSession`) — these are the only fields the record declares. Certificate revocation validation, session/handshake timeouts, and Java KeyStore/TrustStore-based configuration are not present. |
| Default listener | Not Yet Supported | The module-level default listener (`http:defaultListener`) is not implemented. |
| Listener authentication and authorization | Not Yet Supported | `ListenerAuthConfig` and listener-side auth handlers (file user store, LDAP, JWT, OAuth2) are not implemented. |

### Service

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| HTTP service definition and routing | Supported | Path-based routing dispatches to the first attached service whose base path matches, not longest-prefix match; attach overlapping base paths on the same listener in most-specific-first order. |
| Resource function dispatch | Partially Supported | Path parameters (typed segment coercion) and accessor methods (`get`, `post`, `put`, `patch`, `delete`, `head`, `options`, `default`) dispatch to the matching resource function. Resource return values may be `http:Response`, `error`, `()`, or any `anydata` value; an `anydata` return is serialised with the Content-Type its type implies and answered with 201 for a `post` accessor and 200 otherwise, `()` gives 202, and an `error` gives a 500 JSON envelope. Status code response types are not implemented (see the row below), and every error maps to 500 rather than to a status derived from its type. |
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
| XML payloads | Supported | `setXmlPayload()` and `getXmlPayload()` are declared on both `Request` and `Response`, `RequestMessage` admits `xml` (an `xml` message is sent as `application/xml`), `application/xml` and `text/xml` responses bind to `xml` targets, and a resource may return `xml`. There is no `mime:Entity` layer, so multipart XML parts are still out of scope. |

### Notable Behavioural Changes

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
