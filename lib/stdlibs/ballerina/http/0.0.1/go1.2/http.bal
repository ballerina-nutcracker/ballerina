// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Supported subset of ballerina/http for the Go runtime.
// See lib/http/client-support.md for the full feature support matrix.

// ── Shared types ─────────────────────────────────────────────────────────────

// Represents the parsed header value details.
// Fields: value - the header value; params - map of header parameters.
public type HeaderValue record {|
    string value;
    map<string> params;
|};

# Parses the header value which contains multiple values or parameters.
# ```ballerina
#  http:HeaderValue[] values = check http:parseHeader("text/plain;level=1;q=0.6, application/xml;level=2");
# ```
#
# + headerValue - The header value
# + return - An array of `http:HeaderValue` typed records containing the value and its parameter map,
#            or an `error` if the header parsing fails
public isolated function parseHeader(string headerValue) returns HeaderValue[]|error = external;

// ── TLS / secure-socket types ─────────────────────────────────────────────────

// Represents a combination of certificate and private key for mutual TLS.
// The key file must be an unencrypted PEM file; password-protected keys are not supported.
// Fields: certFile - path to the certificate chain; keyFile - path to the private key;
//         keyPassword - accepted at compile time, ignored at runtime.
public type CertKey record {|
    string certFile;
    string keyFile;
    string keyPassword?;
|};

// SSL/TLS protocol type. "SSL" and "DTLS" are accepted at compile time but have no effect
// at runtime — the Go TLS stack only supports TLS.
public type Protocol "SSL"|"TLS"|"DTLS";

// Provides configuration for TLS protocol version constraints.
// Fields: name - the SSL protocol ("SSL", "TLS", or "DTLS");
//         versions - list of enabled protocol versions (e.g., ["TLSv1.2", "TLSv1.3"]).
public type ProtocolConfig record {|
    Protocol name;
    string[] versions;
|};

// Certificate validation type. Accepted at compile time; not implemented at runtime
// (OCSP/CRL checks are not supported in Go's standard TLS library).
public type CertValidationType "OCSP_CRL"|"OCSP_STAPLING";

// Provides configuration for certificate revocation validation.
// Accepted at compile time for compatibility; not enforced at runtime.
// Fields: 'type - certificate validation type; cacheSize - maximum cache size;
//         cacheValidityPeriod - cache entry validity period.
public type CertValidation record {|
    CertValidationType 'type;
    int cacheSize;
    int cacheValidityPeriod;
|};

// Provides configurations for facilitating secure communication with a remote HTTP endpoint.
//
// Supported: enable, verifyHostName, cert (PEM path), key (CertKey), serverName,
//            ciphers, handshakeTimeout, shareSession, protocol.versions.
// Not supported: cert as crypto:TrustStore, key as crypto:KeyStore, certValidation,
//               sessionTimeout, protocol.name (Go supports TLS only).
//
// Fields:
//   enable         - Enable SSL validation. Set false to skip certificate verification.
//   cert           - PEM file path for custom CA trust store (crypto:TrustStore not supported).
//   key            - Mutual TLS certificate+key pair (crypto:KeyStore not supported).
//   protocol       - TLS protocol and version constraints.
//   certValidation - OCSP/CRL settings (accepted; not enforced at runtime).
//   ciphers        - IANA cipher suite names for TLS 1.2; unknown names silently skipped.
//   verifyHostName - Enable/disable hostname verification.
//   shareSession   - Enable/disable TLS session ticket reuse.
//   handshakeTimeout - Maximum TLS handshake duration (seconds).
//   sessionTimeout - TLS session timeout (accepted; not configurable in Go).
//   serverName     - SNI override; defaults to hostname from the target URL.
public type ClientSecureSocket record {|
    boolean enable = true;
    string cert?;
    CertKey key?;
    ProtocolConfig? protocol?;
    CertValidation? certValidation?;
    string[] ciphers?;
    boolean verifyHostName = true;
    boolean shareSession = true;
    decimal handshakeTimeout?;
    decimal sessionTimeout?;
    string serverName?;
|};

// ── Client configuration ──────────────────────────────────────────────────────

// Provides configurations for response size validation.
//
// Notes:
//   maxStatusLineLength - Accepted for API compatibility; not enforced at runtime (no Go transport
//                         equivalent). jBallerina enforces this via Netty's HttpClientCodec.
//   maxHeaderSize       - Maps to Go's http.Transport.MaxResponseHeaderBytes. If the response
//                         headers exceed this limit the request returns an error.
//   maxEntityBodySize   - Enforced per-response via a counting reader. -1 = no limit.
//                         Unlike jBallerina (Netty), streaming enforcement is not applied when
//                         the server omits Content-Length and sends less than maxHeaderSize bytes
//                         before the limit reader fires; the error surfaces on payload extraction.
//
// Fields:
//   maxStatusLineLength - Maximum bytes for the HTTP status line (default: 4096).
//   maxHeaderSize       - Maximum bytes for all response headers combined (default: 8192).
//   maxEntityBodySize   - Maximum bytes for the response body; -1 = no limit (default: -1).
public type ResponseLimitConfigs record {|
    int maxStatusLineLength = 4096;
    int maxHeaderSize = 8192;
    int maxEntityBodySize = -1;
|};

// Provides configurations for connecting to a proxy server.
// The proxy is enabled by setting a non-empty `host`.
//
// Fields:
//   host     - Proxy server hostname (default: ""; empty = no proxy).
//   port     - Proxy server port (default: 0).
//   userName - Proxy auth username (default: ""; empty = no auth).
//   password - Proxy auth password (default: "").
public type ProxyConfig record {|
    string host = "";
    int port = 0;
    string userName = "";
    string password = "";
|};

// Provides configurations for controlling the behaviour in response to HTTP redirect responses
// (status codes 301, 302, 303, 307, 308).
// Fields: enabled - enable/disable redirect following (default: false);
//         maxCount - maximum redirects to follow (default: 5);
//         allowAuthHeaders - forward Authorization headers on redirect (default: false).
public type FollowRedirects record {|
    boolean enabled = false;
    int maxCount = 5;
    boolean allowAuthHeaders = false;
|};

// Provides a set of configurations for controlling the connection pooling behaviour.
// Defaults mirror jBallerina's PoolConfiguration:
//   maxActiveConnections=-1 (unlimited), maxIdleConnections=100, waitTime=30s.
//
// Fields:
//   maxActiveConnections         - Maximum number of active connections the client pool can create
//                                  per endpoint. -1 means unlimited.
//   maxIdleConnections           - Maximum number of idle connections the client pool can hold
//                                  per endpoint.
//   waitTime                     - Maximum time (in seconds) a request will wait to acquire an
//                                  idle connection before erroring.
//   maxActiveStreamsPerConnection - Maximum active streams per HTTP/2 connection (HTTP/2 only).
public type PoolConfiguration record {|
    int maxActiveConnections = -1;
    int maxIdleConnections = 100;
    decimal waitTime = 30;
    int maxActiveStreamsPerConnection = 100;
|};

// HTTP protocol version enum.
// HTTP/1.0 is accepted at compile time; at runtime a warning is printed and HTTP/1.1 is used instead.
public enum HttpVersion {
    HTTP_1_0 = "1.0",
    HTTP_1_1 = "1.1",
    HTTP_2_0 = "2.0"
}

// Compression negotiation options for the HTTP client.
// Controls the Accept-Encoding header on outbound requests and transparent
// decompression of Content-Encoding: gzip/deflate response bodies.
public enum Compression {
    // No Accept-Encoding header is added; the server decides whether to compress.
    COMPRESSION_AUTO = "AUTO",
    // Adds Accept-Encoding: deflate, gzip to outbound requests if not already set.
    // Compressed responses are transparently decompressed.
    COMPRESSION_ALWAYS = "ALWAYS",
    // Removes any Accept-Encoding header, asking the server not to compress the response.
    COMPRESSION_NEVER = "NEVER"
}

// Provides a set of configurations for controlling the behaviours when communicating with
// a remote HTTP endpoint.
//
// Supported: timeout, httpVersion, followRedirects, secureSocket, poolConfig, compression,
//            responseLimits, proxy.
// Not supported: circuitBreaker, retryConfig, cookieConfig, cache,
//               auth, http1Settings, http2Settings, socketConfig,
//               validation, laxDataBinding.
//
// Fields:
//   timeout         - Max wait time in seconds before request times out (default: 30).
//   followRedirects - Redirect handling configuration; () disables redirect following.
//   httpVersion     - HTTP protocol version: HTTP_1_1 or HTTP_2_0 (default).
//   secureSocket    - TLS settings; () uses default TLS verification.
//   poolConfig      - Connection pool settings; () uses platform defaults
//                     (maxIdleConnections=100, maxActiveConnections=-1, waitTime=30s).
//   compression     - Compression negotiation mode (default: COMPRESSION_AUTO).
//                     COMPRESSION_NEVER disables Accept-Encoding and response decompression.
//   responseLimits  - Response size limits (default: maxStatusLineLength=4096,
//                     maxHeaderSize=8192, maxEntityBodySize=-1).
//   proxy           - HTTP proxy configuration; () disables proxy.
public type ClientConfiguration record {|
    decimal timeout = 30;
    FollowRedirects? followRedirects = ();
    HttpVersion httpVersion = HTTP_2_0;
    ClientSecureSocket? secureSocket = ();
    PoolConfiguration? poolConfig = ();
    Compression compression = COMPRESSION_AUTO;
    ResponseLimitConfigs responseLimits = {};
    ProxyConfig? proxy = ();
|};

// ── Header position ───────────────────────────────────────────────────────────

// Represents the position of an HTTP header: leading (transport) or trailing (after body).
// TRAILING is accepted at compile time but all runtime operations act on transport headers.
public type HeaderPosition "LEADING"|"TRAILING";

# Represents the leading header position (transport headers). This is the default for all
# header operations.
public const HeaderPosition LEADING = "LEADING";

# Represents the trailing header position. Accepted at compile time for API compatibility;
# at runtime all header operations act on transport (leading) headers.
public const HeaderPosition TRAILING = "TRAILING";

// ── Response ──────────────────────────────────────────────────────────────────

# Represents an HTTP response.
#
# `Response` objects are created by the HTTP client after a successful request.
# They can also be constructed explicitly using `new http:Response()` and populated
# with `setTextPayload`, `setJsonPayload`, `setBinaryPayload`, and `setHeader`
# before being returned.
public class Response {
    # The HTTP status code (e.g., 200, 404, 500).
    public int statusCode = 200;
    # The response reason phrase (e.g., "OK").
    public string reasonPhrase = "";
    # The server header value.
    public string server = "";
    # The resolved URI after any redirects.
    public string resolvedRequestedURI = "";

    # Initialises the response with status code 200 and empty headers and body.
    public isolated function init() {
        self.initNative();
    }

    private isolated function initNative() = external;

    # Sets the response body to a plain string and Content-Type to `text/plain`.
    #
    # + payload     - The string payload
    # + contentType - Optional MIME type; defaults to `text/plain`
    public isolated function setTextPayload(string payload, string? contentType = ()) = external;

    # Sets the response body to a JSON value and Content-Type to `application/json`.
    #
    # + payload     - The JSON payload
    # + contentType - Optional MIME type; defaults to `application/json`
    public isolated function setJsonPayload(json payload, string? contentType = ()) = external;

    # Sets the response body to a byte array and Content-Type to `application/octet-stream`.
    #
    # + payload     - The binary payload
    # + contentType - Optional MIME type; defaults to `application/octet-stream`
    public isolated function setBinaryPayload(byte[] payload, string? contentType = ()) = external;

    # Sets or replaces a response header.
    #
    # + headerName  - The header name (case-insensitive)
    # + headerValue - The header value
    public isolated function setHeader(string headerName, string headerValue) = external;

    # Adds a value to the specified header without replacing existing values.
    #
    # + headerName  - The header name (case-insensitive)
    # + headerValue - The header value to add
    # + position    - Header position (`LEADING` or `TRAILING`); `TRAILING` is accepted but ignored
    public isolated function addHeader(string headerName, string headerValue, HeaderPosition position = LEADING) = external;

    # Removes the specified header from the response.
    #
    # + headerName - The header name (case-insensitive)
    # + position   - Header position (`LEADING` or `TRAILING`); `TRAILING` is accepted but ignored
    public isolated function removeHeader(string headerName, HeaderPosition position = LEADING) = external;

    # Removes all headers from the response.
    #
    # + position - Header position (`LEADING` or `TRAILING`); `TRAILING` is accepted but ignored
    public isolated function removeAllHeaders(HeaderPosition position = LEADING) = external;

    # Sets the `Content-Type` header on the response.
    #
    # + contentType - The MIME type string
    # + return - `()` on success, or an `error` if the content type is invalid
    public isolated function setContentType(string contentType) returns error? = external;

    # Returns the `Content-Type` header value of the response.
    #
    # + return - The content type string, or `""` if not set
    public isolated function getContentType() returns string = external;

    # Returns the response body as a plain string.
    #
    # + return - The response body as a `string`
    public isolated function getTextPayload() returns string = external;

    # Parses the response body as JSON.
    #
    # + return - The parsed `json` value, or an `error` if the body is not valid JSON
    public isolated function getJsonPayload() returns json|error = external;

    # Returns the response body as a byte array.
    #
    # + return - The response body as `byte[]`, or an `error` if extraction fails
    public isolated function getBinaryPayload() returns byte[]|error = external;

    # Checks whether the specified header is present in the response.
    #
    # + headerName - The header name (case-insensitive)
    # + position - Header position (`LEADING` or `TRAILING`). `TRAILING` is accepted
    #              but all lookups operate on transport headers
    # + return - `true` if the header exists, `false` otherwise
    public isolated function hasHeader(string headerName, HeaderPosition position = LEADING) returns boolean = external;

    # Returns the first value for the specified header.
    #
    # + headerName - The header name (case-insensitive)
    # + position - Header position (`LEADING` or `TRAILING`). `TRAILING` is accepted
    #              but all lookups operate on transport headers
    # + return - The first header value, or an `error` if the header is not found
    public isolated function getHeader(string headerName, HeaderPosition position = LEADING) returns string|error = external;

    # Returns all values for the specified header.
    #
    # + headerName - The header name (case-insensitive)
    # + position - Header position (`LEADING` or `TRAILING`). `TRAILING` is accepted
    #              but all lookups operate on transport headers
    # + return - A `string[]` of all values for the header, or an `error` if the header is not found
    public isolated function getHeaders(string headerName, HeaderPosition position = LEADING) returns string[]|error = external;

    # Returns the names of all response headers.
    #
    # + position - Header position (`LEADING` or `TRAILING`). `TRAILING` is accepted
    #              but all lookups operate on transport headers
    # + return - An array of all header names present in the response
    public isolated function getHeaderNames(HeaderPosition position = LEADING) returns string[] = external;
}

// ── Request ───────────────────────────────────────────────────────────────────

# Represents an HTTP request. Used both for constructing outbound requests (client-side `forward`)
# and for inspecting inbound requests in resource functions.
#
# To construct an outbound request, create a `new http:Request()`, set the `method` field,
# and call `setTextPayload`, `setJsonPayload`, `setBinaryPayload`, or `setHeader` to populate it.
public class Request {
    # The raw URI path of the request (including query string if present).
    public string rawPath = "";
    # The HTTP method of the request (e.g., "GET", "POST").
    public string method = "";
    # The HTTP protocol version of the request (e.g., "HTTP/1.1").
    public string httpVersion = "";

    # Initialises the request with an empty body and headers.
    public isolated function init() {
        self.initNative();
    }

    private isolated function initNative() = external;

    # The user agent value for the request.
    public string userAgent = "";
    # The extra path info for the request.
    public string extraPathInfo = "";

    # Sets the request body as plain text with `Content-Type: text/plain`.
    #
    # + payload     - The text body to set
    # + contentType - Optional MIME type; defaults to `text/plain`
    public isolated function setTextPayload(string payload, string? contentType = ()) = external;

    # Sets the request body as JSON with `Content-Type: application/json`.
    #
    # + payload     - The JSON value to set
    # + contentType - Optional MIME type; defaults to `application/json`
    public isolated function setJsonPayload(json payload, string? contentType = ()) = external;

    # Sets the request body as bytes with `Content-Type: application/octet-stream`.
    #
    # + payload     - The byte array to set
    # + contentType - Optional MIME type; defaults to `application/octet-stream`
    public isolated function setBinaryPayload(byte[] payload, string? contentType = ()) = external;

    # Sets a header on the request. Replaces any existing value for the header.
    #
    # + headerName  - The header name
    # + headerValue - The header value
    public isolated function setHeader(string headerName, string headerValue) = external;

    # Adds a value to the specified header without replacing existing values.
    #
    # + headerName  - The header name (case-insensitive)
    # + headerValue - The header value to add
    public isolated function addHeader(string headerName, string headerValue) = external;

    # Removes the specified header from the request.
    #
    # + headerName - The header name (case-insensitive)
    public isolated function removeHeader(string headerName) = external;

    # Removes all headers from the request.
    public isolated function removeAllHeaders() = external;

    # Returns the names of all request headers.
    #
    # + return - An array of all header names present in the request
    public isolated function getHeaderNames() returns string[] = external;

    # Sets the `Content-Type` header on the request.
    #
    # + contentType - The MIME type string
    # + return - `()` on success, or an `error` if the content type is invalid
    public isolated function setContentType(string contentType) returns error? = external;

    # Returns the `Content-Type` header value of the request.
    #
    # + return - The content type string, or `""` if not set
    public isolated function getContentType() returns string = external;

    # Returns the request body as a plain string.
    #
    # + return - The request body as a `string`, or an `error` if extraction fails
    public isolated function getTextPayload() returns string|error = external;

    # Parses the request body as JSON.
    #
    # + return - The parsed `json` value, or an `error` if the body is not valid JSON
    public isolated function getJsonPayload() returns json|error = external;

    # Returns the request body as a byte array.
    #
    # + return - The request body as `byte[]`, or an `error` if extraction fails
    public isolated function getBinaryPayload() returns byte[]|error = external;

    # Returns the first value for the specified request header.
    #
    # + headerName - The header name (case-insensitive)
    # + return - The first header value, or an `error` if the header is not found
    public isolated function getHeader(string headerName) returns string|error = external;

    # Returns all values for the specified request header.
    #
    # + headerName - The header name (case-insensitive)
    # + return - A `string[]` of all values for the header, or an `error` if not found
    public isolated function getHeaders(string headerName) returns string[]|error = external;

    # Checks whether the specified header is present in the request.
    #
    # + headerName - The header name (case-insensitive)
    # + return - `true` if the header exists, `false` otherwise
    public isolated function hasHeader(string headerName) returns boolean = external;

    # Returns all query parameters as a map of string arrays.
    #
    # + return - A `map<string[]>` of query parameter names to value lists
    public isolated function getQueryParams() returns map<string[]> = external;

    # Returns the first value of a query parameter by name.
    #
    # + paramName - The query parameter name
    # + return - The first value as a `string`, or `()` if the parameter is not present
    public isolated function getQueryParamValue(string paramName) returns string? = external;

    # Returns all values for the specified query parameter.
    #
    # + key - The query parameter name
    # + return - A `string[]` of all values, or `()` if the parameter is not present
    public isolated function getQueryParamValues(string key) returns string[]? = external;
}

// Represents an HTTP outbound request entity. Accepts any JSON-compatible value or an
// existing `Request` object. The `mime:Entity[]` and `stream` subtypes of jBallerina's
// `RequestMessage` are not supported in this implementation.
public type RequestMessage json|Request;

// Represents HTTP methods.
public enum Method {
    GET,
    POST,
    PUT,
    DELETE,
    PATCH,
    HEAD,
    OPTIONS
}

// ── Client ────────────────────────────────────────────────────────────────────

# The HTTP client provides functionality to connect to remote HTTP services and perform
# requests using the standard HTTP methods GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS,
# EXECUTE, and FORWARD.
#
# **Supported methods:** `get`, `post`, `put`, `patch`, `delete`, `head`, `options`,
# `execute`, `forward`.
#
# **Not supported:** `submit`/`getResponse`, HTTP/2 server push methods
# (`hasPromise`, `getNextPromise`, etc.), and resource function syntax (`client->/path.get(...)`).
#
# **Message body:** The `message` parameter accepts `RequestMessage` (`json|Request`).
# The `Content-Type` is inferred when not explicitly overridden via `mediaType`:
# - `string` → `text/plain`
# - `byte[]` → `application/octet-stream`
# - `http:Request` → content type and body taken from the request
# - everything else → serialised as `application/json`
#
# The `mediaType` parameter overrides the inferred `Content-Type` in all cases.
#
# **Return type:** All methods return `Response|error`. Automatic data binding via
# `targetType` is not supported — use `getTextPayload()`, `getJsonPayload()`, or
# `getBinaryPayload()` to extract the response body.
#
# **Error types:** Errors are plain `error` values. The upstream distinct error types
# (`http:ClientError`, `http:HeaderNotFoundError`, etc.) are not declared.
public isolated client class Client {

    # Gets invoked to initialize the `client`. During initialization, the configurations
    # provided through the `config` record control timeout, TLS, HTTP version, redirect
    # behaviour, and connection pooling.
    #
    # + url - The base URL of the target service
    # + config - The configurations to be used when initializing the `client`.
    #            Unsupported fields (`circuitBreaker`, `retryConfig`, `cookieConfig`, etc.)
    #            are not available in this implementation
    # + return - `()` on success, or an `error` if initialisation fails
    public isolated function init(string url, ClientConfiguration config = {}) returns error? {
        return self.initNative(url, config);
    }

    private isolated function initNative(string url, ClientConfiguration config) returns error? = external;

    # Retrieves a representation of the specified resource from the remote HTTP endpoint.
    #
    # + path - The request path (appended to the base URL)
    # + headers - Optional request headers as a `map<string|string[]>`
    # + return - The `http:Response` or an `error` if the request fails
    remote isolated function get(string path, map<string|string[]>? headers = ()) returns Response|error = external;

    # Creates a new resource or submits data to a resource for processing.
    #
    # + path - The request path (appended to the base URL)
    # + message - The request body (`string`, `byte[]`, JSON-compatible value, or `http:Request`)
    # + headers - Optional request headers as a `map<string|string[]>`
    # + mediaType - Optional `Content-Type` override; inferred from `message` if omitted
    # + return - The `http:Response` or an `error` if the request fails
    remote isolated function post(string path, RequestMessage message, map<string|string[]>? headers = (),
            string? mediaType = ()) returns Response|error = external;

    # Creates a new resource or replaces a representation of the specified resource.
    #
    # + path - The request path (appended to the base URL)
    # + message - The request body (`string`, `byte[]`, JSON-compatible value, or `http:Request`)
    # + headers - Optional request headers as a `map<string|string[]>`
    # + mediaType - Optional `Content-Type` override; inferred from `message` if omitted
    # + return - The `http:Response` or an `error` if the request fails
    remote isolated function put(string path, RequestMessage message, map<string|string[]>? headers = (),
            string? mediaType = ()) returns Response|error = external;

    # Applies a partial modification to the specified resource.
    #
    # + path - The request path (appended to the base URL)
    # + message - The request body (`string`, `byte[]`, JSON-compatible value, or `http:Request`)
    # + headers - Optional request headers as a `map<string|string[]>`
    # + mediaType - Optional `Content-Type` override; inferred from `message` if omitted
    # + return - The `http:Response` or an `error` if the request fails
    remote isolated function patch(string path, RequestMessage message, map<string|string[]>? headers = (),
            string? mediaType = ()) returns Response|error = external;

    # Deletes the specified resource.
    #
    # + path - The request path (appended to the base URL)
    # + message - Optional request body (`string`, `byte[]`, JSON-compatible value, or `http:Request`)
    # + headers - Optional request headers as a `map<string|string[]>`
    # + mediaType - Optional `Content-Type` override; inferred from `message` if omitted
    # + return - The `http:Response` or an `error` if the request fails
    remote isolated function delete(string path, RequestMessage? message = (), map<string|string[]>? headers = (),
            string? mediaType = ()) returns Response|error = external;

    # Requests headers from the specified resource without fetching the response body.
    # Identical to `get` but the server must not return a message body.
    #
    # + path - The request path (appended to the base URL)
    # + headers - Optional request headers as a `map<string|string[]>`
    # + return - The `http:Response` or an `error` if the request fails
    remote isolated function head(string path, map<string|string[]>? headers = ()) returns Response|error = external;

    # Requests the communication options available for the specified resource.
    #
    # + path - The request path (appended to the base URL)
    # + headers - Optional request headers as a `map<string|string[]>`
    # + return - The `http:Response` or an `error` if the request fails
    remote isolated function options(string path, map<string|string[]>? headers = ()) returns Response|error = external;

    # Sends an HTTP request with an explicit verb to the specified path.
    # Use this for HTTP methods not covered by the dedicated remote functions.
    #
    # + httpVerb - The HTTP method to use (e.g., `"GET"`, `"POST"`, `"PUT"`)
    # + path - The request path (appended to the base URL)
    # + message - The request body (`string`, `byte[]`, JSON-compatible value, or `http:Request`)
    # + headers - Optional request headers as a `map<string|string[]>`
    # + mediaType - Optional `Content-Type` override; inferred from `message` if omitted
    # + return - The `http:Response` or an `error` if the request fails
    remote isolated function execute(string httpVerb, string path, RequestMessage message,
            map<string|string[]>? headers = (), string? mediaType = ()) returns Response|error = external;

    # Forwards the inbound `Request` to the specified path, preserving the original HTTP method,
    # headers, and body. Useful for proxy and gateway patterns where the incoming request must be
    # relayed to an upstream service without modification.
    #
    # + path    - The request path (appended to the base URL)
    # + request - The inbound `http:Request` whose method, headers, and body are forwarded
    # + return  - The `http:Response` from the upstream service, or an `error` if the request fails
    remote isolated function forward(string path, Request request) returns Response|error = external;
}
