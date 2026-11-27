# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the `http` error type hierarchy, status code responses, error responses, resource signature parameter binding, responding through a caller, service configuration and CORS, the first steps in completing the `http` service and listener surface.

## [http](https://github.com/ballerina-platform/module-ballerina-http/blob/master/docs/spec/spec.md)

### Error types

The `http:Error` hierarchy from jBallerina is declared in full, along with the `http.httpscerr` module, which provides the status code error types (`httpscerr:StatusCodeError`, `httpscerr:'4XXStatusCodeError`, `httpscerr:NotFoundError`, and so on).

```ballerina
import ballerina/http;
import ballerina/http.httpscerr;
```

| Surface | Error type returned | Notes |
|---|---|---|
| `Request` and `Response` `getTextPayload`, `getJsonPayload`, `getXmlPayload`, `getBinaryPayload` | `http:ClientError` | `http:NoContentError` (`No content`) from the text, JSON and XML accessors when the message has no payload, or when a `json` payload is empty (`getBinaryPayload` returns an empty `byte[]`); `http:GenericClientError` (`Error occurred while retrieving the <kind> payload from the <request\|response>`) when the body cannot be read or decoded |
| `Request` and `Response` `getHeader`, `getHeaders` | `http:HeaderNotFoundError` | `Http header does not exist` |
| `Request.getFormParams` | `http:ClientError` | `http:GenericClientError` for a missing or wrong `Content-Type`; `http:ClientError` for a body that cannot be decoded |
| `Request` and `Response` `getBodyParts` | `http:ClientError` | `http:GenericClientError` with the `mime` error as its cause |
| `parseHeader` | `http:ClientError` | `http:GenericClientError` (`failed to parse: ...`) |
| `Listener.init` | `http:ListenerError` | `http:GenericListenerError`; a missing server certificate or key file reports `Key file or server certificates file not found`, and a certificate, key or mutual TLS CA file without a usable PEM certificate or private key reports `File does not contain valid certificates: <path>` or `File does not contain valid private key: <path>` |
| `Listener` `attach`, `detach`, `'start`, `gracefulStop`, `immediateStop` | `error` | Declared `error?` as in jBallerina; a failure is an `http:GenericListenerError` |

The `Client` remote and resource methods still return plain `error` values; the client error subtypes are tracked with the rest of the client completion.

### Status code responses

A resource can return a status code response record, including a record type that includes one:

```ballerina
type AlbumNotFound record {|
    *http:NotFound;
    Album body;
|};

resource function get albums/[string id]() returns http:Ok|AlbumNotFound {
    // ...
}
```

| Surface | Notes |
|---|---|
| `http:StatusCodeResponse` record types (`http:Ok`, `http:Created`, `http:NotFound`, and the other 58) and `http:DefaultStatusCodeResponse` | Each record defaults `status` to its `Status` object, so `http:Ok ok = {body: "done"};` needs no status |
| `http:Status`, the `Status<Name>` classes, `http:DefaultStatus` | `new http:DefaultStatus(418)` gives a status for any code |
| `STATUS_*` constants | The 61 status code integers, for example `http:STATUS_NOT_FOUND` |
| Returning a record from a resource | Sent with `status.code`, its `headers` (one header per element of an array value), and its `body`: `xml`, `string`, `byte[]` as themselves, a `map<string>` url-encoded when `mediaType` is `application/x-www-form-urlencoded`, anything else as JSON. The `mediaType` field overrides a `Content-Type` in `headers`, which overrides the body's default. A JSON body keeps its members' order and writes nested `xml` as a string. A body that cannot be encoded, or a status that cannot end a response (outside 100–999, or a 1xx other than 101), gives a 500 |

Not yet supported: the `readonly &` response constants (`http:OK`, `http:NOT_FOUND`, ...), the `@http:Payload` return media type, and `@http:Header` renaming of `headers` fields.

### Error responses

A resource that returns an `error` answers with the status its `http.httpscerr` type implies and jBallerina's error body:

```ballerina
resource function get albums/[string id]() returns Album|error {
    return error httpscerr:NotFoundError("no album " + id);
}
```

| Returned error | Response |
|---|---|
| `http:ApplicationResponseError` (`http:ClientRequestError`, `http:RemoteServerError`) | Its `statusCode` detail |
| An `httpscerr` status code error (`httpscerr:NotFoundError`, `httpscerr:ConflictError`, ...) | Its status code |
| `httpscerr:DefaultStatusCodeError` | Its `statusCode` detail, or else its cause's status, or else 500 |
| Any other error | 500 |
| With a `body` detail (`httpscerr` and `http:ApplicationResponseError` errors) | The body, sent as a status code response body is |
| A `headers` detail (`httpscerr` and `http:ApplicationResponseError` errors) | Added, with or without a `body` |
| Without a `body` detail, or any other error (its detail is ignored) | An `http:ErrorPayload` JSON: `timestamp`, `status`, `reason`, `message`, `path` (with the query string), `method` |

Each returned error is logged as `unhandled error returned from the service`. A request that matches no service or resource answers 404, and one whose method matches no resource answers 405, with the same `http:ErrorPayload` body. An `OPTIONS` request also answers 405 when no `options` or `default` resource matches it.

### Resource signature parameters

Resource functions may now declare query, header, and payload parameters alongside path parameters and an `http:Request` parameter, in any order. A binding failure is answered with 400 and a JSON error envelope whose message follows jBallerina (`no query param value found for 'id'`, `header binding failed for parameter: 'x-num'`, `data binding failed: ...`).

| Parameter | Notes |
|---|---|
| Query parameter | Any unannotated parameter that is not the payload, or one annotated `@http:Query {name?}`. Types: `string`, `int`, `float`, `decimal`, `boolean`, `map<anydata>` and records (sent as JSON), arrays of these, and nilable forms. Values are split on commas and the first value binds to a non-array parameter. A missing parameter binds to `()` when nilable; a defaultable parameter does not receive its default and yields 400 instead |
| `@http:Header {name?}` | Types: `string`, `int`, `float`, `decimal`, `boolean`, arrays of these, nilable forms, and closed records whose field names are the header names. An absent or empty header binds to `()` when nilable |
| `@http:Payload {mediaType?}` | Any `anydata` type, deserialised by the builder the `Content-Type` selects (JSON, XML, text, form-urlencoded, octet-stream) or by the parameter type when the media type is absent or unknown. An empty body binds to `()` when nilable. `mediaType` is accepted but not enforced, as in jBallerina |
| Default payload parameter | In a resource other than `get`, `head`, and `options`, the single unannotated structured parameter (a record, map, table, tuple, structured array, `byte[]`, or `xml`) is the payload |

Signatures jBallerina's compiler plugin rejects — an ambiguous default payload, an unsupported parameter type, or two `http:Request` parameters — are reported per request with 500 instead.

```ballerina
type Album record {|
    string title;
    int year;
|};

service /albums on new http:Listener(9090) {
    resource function get search(string artist, int? count, @http:Header {name: "X-Trace"} string? trace) returns string {
        return string `${artist} ${count.toString()} ${trace ?: "-"}`;
    }

    resource function post .(Album album) returns Album {
        return album;
    }
}
```

## Responding through a caller

A resource can take an `http:Caller` parameter and send its response with `respond`, instead of returning it.

```ballerina
import ballerina/http;

service /albums on new http:Listener(9090) {
    resource function post .(http:Caller caller, @http:Payload json album) returns error? {
        check caller->respond({id: 1, album});
    }
}
```

| Case | Behaviour |
|---|---|
| `respond(message)` | Sent like a returned value of the same type: an `http:Response` as is, a status code record with its status, an `error` with its status code error response, `string`/`xml`/`byte[]`/`mime:Entity[]` with their content type and any other `anydata` as JSON; a payload from a `post` resource answers 201 |
| A second `respond`, or one after the resource returned | Returns `http:GenericListenerError` `illegal respond: response has already been sent` |
| The resource returns `()` without responding | 500 with no body |
| The resource returns an error without responding | The error's response, as for a returned error |
| The resource returns an error after responding | Nothing more is sent; stderr gets `illegal return: response has already been sent` |
| `redirect(response, code, locations)` | Sends `response` with `code` and a `Location` header joining `locations` with `,` |
| `'continue()` | Sends an interim 100 Continue; the exchange stays open for `respond` |

A resource with an `http:Caller` parameter may only return `error` or nil, and may declare one `http:Caller` parameter; the compiler plugin reports anything else.

## Configuring a service

`@http:ServiceConfig` configures how a service is dispatched and how its parameters and responses are handled.

```ballerina
import ballerina/http;

type Album record {|
    string title;
    string? artist;
|};

@http:ServiceConfig {
    host: "music.example.com",
    treatNilableAsOptional: false,
    laxDataBinding: true,
    mediaTypeSubtypePrefix: "vnd.music"
}
service /albums on new http:Listener(9090) {
    resource function post .(Album album) returns Album => album;
}
```

| Field | Behaviour |
|---|---|
| `host` | The service answers only the requests whose `Host` header names that host, compared case-insensitively; without a port it matches any port, and a service for the exact host and port takes precedence. Services without a `host` answer the other requests, and services on different hosts may share a base path |
| `treatNilableAsOptional: false` | A missing query parameter or header gives 400 even when its type is nilable; a value-less query parameter or an empty header still binds `()`, and a nilable header record parameter binds `()` when one of its headers is missing |
| `laxDataBinding: true` | A JSON body bound to a record parameter, not a nilable record, a map or an array, drops the fields a closed record lacks, removes a `null` optional field whose type does not admit `()` (a nilable one keeps its `null`), and binds `()` for an absent required nilable field; an absent optional field stays absent. A record with only a rest field, such as `record {\| T...; \|}`, is the same type as `map<T>` and is bound strictly like a map, where jBallerina projects it |
| `compression` | Responses are compressed with gzip or deflate when the request's `Accept-Encoding` asks for it; `COMPRESSION_ALWAYS` also compresses when it does not, `COMPRESSION_NEVER` turns this off, and `contentTypes` limits compression to the listed media types; a `Content-Encoding: gzip` or `deflate` the resource sets still compresses the body |
| `chunking` | Over HTTP/1.1, `CHUNKING_AUTO` chunks a body over 8192 bytes, `CHUNKING_ALWAYS` always chunks and `CHUNKING_NEVER` sends a `Content-Length` |
| `mediaTypeSubtypePrefix` | Every `Content-Type` a resource of the service sends, including its binding and error responses, gets the prefix on its subtype, for example `application/vnd.music+json`; a subtype that already starts with the prefix and a `multipart` type are left unchanged |

The compiler plugin rejects a `mediaTypeSubtypePrefix` that is not a valid subtype or has a `+` suffix, and a `basePath` on a service declaration. `@http:ResourceConfig` is declared but has no effect yet, and so is `@http:Cache` on a resource's return type, which adds no cache headers until return-type annotations can be read at runtime ([#1355](https://github.com/ballerina-nutcracker/ballerina/issues/1355)). The compiler plugin rejects `@http:Cache` and `@http:Payload` on a return type that can only be an error or nil.

## Cross-origin resource sharing

The `cors` of `@http:ServiceConfig` answers browsers' cross-origin requests for every resource of the service, and an `OPTIONS` request that no resource serves is answered with the methods the path serves.

```ballerina
import ballerina/http;

@http:ServiceConfig {
    cors: {
        allowOrigins: ["https://music.example.com"],
        allowCredentials: true,
        allowHeaders: ["X-Request-Id"],
        maxAge: 600
    }
}
service /albums on new http:Listener(9090) {
    resource function get .() returns string[] => ["Blue Train"];

    resource function post .(@http:Payload string title) returns string => title;
}
```

| Request | Response |
|---|---|
| A request with an allowed `Origin` | The resource's response with `Access-Control-Allow-Origin` (the request's origin), `Vary: Origin`, and `Access-Control-Allow-Credentials` and `Access-Control-Expose-Headers` when `allowCredentials` and `exposeHeaders` are set; credentials are allowed only for origins `allowOrigins` lists |
| A request with another `Origin` | The resource's response without CORS headers |
| `OPTIONS` with one allowed `Origin`, an `Access-Control-Request-Method` the path serves and the config allows, and allowed `Access-Control-Request-Headers` | 204 with `Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, the requested `Access-Control-Allow-Headers`, and `Access-Control-Allow-Credentials` and `Access-Control-Max-Age` when configured |
| Any other `OPTIONS` that no resource serves | 204 with `Allow` listing the methods the path serves, or every method of the service at its base path; 404 when the path serves none |

`@http:ResourceConfig {cors}` is not supported yet.

## Request interceptors

An `http:InterceptableService` runs the `http:RequestInterceptor` services its `createInterceptors` method returns before its own resources. An interceptor runs when its single resource matches the request path and method, and passes the request on by returning `ctx.next()`. The interceptors and the resource share one `http:RequestContext`.

```ballerina
import ballerina/http;

service class TenantReader {
    *http:RequestInterceptor;

    resource function 'default [string... path](http:RequestContext ctx, @http:Header string? x\-tenant)
            returns http:BadRequest|http:NextService|error? {
        if x\-tenant is () {
            http:BadRequest badRequest = {body: "missing tenant"};
            return badRequest;
        }
        ctx.set("tenant", x\-tenant);
        return ctx.next();
    }
}

service http:InterceptableService /albums on new http:Listener(9090) {
    public function createInterceptors() returns http:Interceptor[] => [new TenantReader()];

    resource function get .(http:RequestContext ctx) returns string|error {
        string tenant = check ctx.getWithType("tenant");
        return "albums for " + tenant;
    }
}
```

| Interceptor result | Response |
|---|---|
| The stage `ctx.next()` returned | The request goes on to the next matching interceptor, then to the resource; a 404 or 405 for the resource comes after the interceptors |
| Another service | An `http:InterceptorReturnError`, `next interceptor service did not match with the configuration`, or `target service did not match with the configuration` past the last interceptor, which by default answers 500 |
| `()` from an interceptor without an `http:Caller` parameter | 202, and the request goes no further |
| `()` from an interceptor with an `http:Caller` parameter | 500 if it sent no response, and the request goes no further |
| An error | The error goes to the next `http:RequestErrorInterceptor`, or else to the error handling of the response |
| Any other value, `http:Response` or status code response | That response, which the response interceptors before the interceptor (or before the stage its `ctx.next()` reached) receive, with the `@http:ServiceConfig` of the service applied |

`ctx.next()` returns an error from the resource itself or once it has passed the service. The `JWT_INFORMATION` member is not supported yet.

## Response and error interceptors

The `http:ResponseInterceptor` services of an `http:InterceptableService` receive each response through their `interceptResponse` remote method, from the last towards the first. An error skips them for the nearest `http:ResponseErrorInterceptor`, whose `interceptResponseError` receives the error and the response it maps to; an error no interceptor handles gets the default error response. An `http:RequestErrorInterceptor` handles the errors of the request interceptors before it through a `'default [string... path]` resource with an `error` parameter.

```ballerina
import ballerina/http;

service class ServedBy {
    *http:ResponseInterceptor;

    remote function interceptResponse(http:RequestContext ctx, http:Response res) returns http:NextService|error? {
        res.setHeader("x-served-by", "albums");
        return ctx.next();
    }
}

service class ErrorConverter {
    *http:ResponseErrorInterceptor;

    remote function interceptResponseError(error err, http:Response res) returns http:Response {
        http:Response converted = new;
        converted.statusCode = res.statusCode;
        converted.setJsonPayload({message: err.message()});
        return converted;
    }
}

service http:InterceptableService /albums on new http:Listener(9090) {
    public function createInterceptors() returns http:Interceptor[] => [new ErrorConverter(), new ServedBy()];

    resource function get [string id]() returns string|error {
        if id == "0" {
            return error("no album 0");
        }
        return "album " + id;
    }
}
```

`GET /albums/1` answers `album 1` with `x-served-by: albums`; `GET /albums/0` answers 500 `{"message":"no album 0"}`, without the header, since the error skips `ServedBy` for the `ErrorConverter` before it.

| Interceptor result | Response |
|---|---|
| The stage `ctx.next()` returned | The response goes on to the next response interceptor towards the first; once `ctx.next()` has passed the first, `()` sends it as it is |
| Another service | An `http:InterceptorReturnError` for the error interceptors |
| `()` | A 202 replaces the response, and the response interceptors before the interceptor receive it |
| An error | The error goes to the nearest response error interceptor before the interceptor, or else gets the default error response |
| Any other value, `http:Response` or status code response | That response replaces the response, and the response interceptors before the interceptor receive it |

## Default listener

`http:getDefaultListener()` returns one listener shared by every module of the program. It is created on the first call, on port `http:defaultListenerPort` with `http:defaultListenerConfig`.

```ballerina
import ballerina/http;

listener http:Listener albums = http:getDefaultListener();

service /albums on albums {
    resource function get .() returns string[] => ["Blue Train"];
}

service /artists on albums {
    resource function get .() returns string[] => ["John Coltrane"];
}
```

Both services are served on port 9090, which `albums.getPort()` returns, and a `listener` declaration in another module that calls `http:getDefaultListener()` gets the same listener. A service cannot attach to `http:getDefaultListener()` directly, since it may return an error; jBallerina rejects that, but the Go-native compiler does not yet ([#1811](https://github.com/ballerina-nutcracker/ballerina/issues/1811)). `defaultListenerPort` and `defaultListenerConfig` keep jBallerina's defaults, since `Config.toml` cannot set them until configurable variables are supported.

## HTTP/2 server push

Over an HTTP/2 connection, a resource can promise the client a response it has not asked for yet and send it on a stream of its own.

```ballerina
import ballerina/http;

service / on new http:Listener(9090) {
    resource function get page(http:Caller caller) returns error? {
        http:PushPromise styles = new ("/styles.css");
        check caller->promise(styles);
        check caller->respond("<link rel=\"stylesheet\" href=\"/styles.css\">");

        http:Response css = new;
        css.setTextPayload("body { margin: 0; }", "text/css");
        check caller->pushPromisedResponse(styles, css);
    }
}
```

`promise` returns an `http:ListenerError` over HTTP/1.x, when the client has disabled push, and for a promise with a relative or malformed path or a method other than `GET` or `HEAD`; jBallerina sends such promises, and the client then drops the connection. A promise the resource returns without answering is reset, where jBallerina leaves the client waiting for it.

## Listener configuration

`http:ListenerConfiguration` declares jBallerina's connection settings with their defaults.

```ballerina
import ballerina/http;

listener http:Listener albums = new (9090, {
    http1Settings: {keepAlive: http:KEEPALIVE_NEVER},
    http2MaxActiveStreams: 50,
    http2InitialWindowSize: 131072,
    socketConfig: {tcpNoDelay: false, keepAlive: true},
    gracefulStopTimeout: 10
});
```

`KEEPALIVE_NEVER` closes each HTTP/1.x connection after its response, on an HTTP/2 listener too, where jBallerina applies `http1Settings` only to an HTTP/1.1 listener. The HTTP/2 settings are advertised to every HTTP/2 client, over TLS too, where jBallerina ignores `http2InitialWindowSize`. `gracefulStopTimeout` bounds how long `gracefulStop` waits for requests in flight. `KEEPALIVE_ALWAYS` behaves like `KEEPALIVE_AUTO`, following the request's `Connection` header. `maxPipelinedRequests`, `soBackLog`, `socketReuse`, `connectTimeOut` and the stale-eviction settings are accepted but have no effect.
