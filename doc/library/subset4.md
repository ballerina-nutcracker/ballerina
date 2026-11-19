# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the `http` error type hierarchy, status code responses and error responses, the first steps in completing the `http` service and listener surface.

## [http](https://github.com/ballerina-platform/module-ballerina-http/blob/master/docs/spec/spec.md)

### Error types

The `http:Error` hierarchy from jBallerina is declared in full, along with the `http.httpscerr` module, which provides the status code error types (`httpscerr:StatusCodeError`, `httpscerr:'4XXStatusCodeError`, `httpscerr:NotFoundError`, and so on).

```ballerina
import ballerina/http;
import ballerina/http.httpscerr;
```

| Surface | Error type returned | Notes |
|---|---|---|
| `Request` and `Response` `getTextPayload`, `getJsonPayload`, `getXmlPayload`, `getBinaryPayload` | `http:ClientError` | `http:NoContentError` (`No content`) when the message has no payload, or when a `json` payload is empty; `http:GenericClientError` (`Error occurred while retrieving the <kind> payload from the <request\|response>`) when the body cannot be read or decoded |
| `Request` and `Response` `getHeader`, `getHeaders` | `http:HeaderNotFoundError` | `Http header does not exist` |
| `Request.getFormParams` | `http:ClientError` | `http:GenericClientError` for a missing or wrong `Content-Type`; `http:ClientError` for a body that cannot be decoded |
| `Request` and `Response` `getBodyParts` | `http:ClientError` | `http:GenericClientError` with the `mime` error as its cause |
| `parseHeader` | `http:ClientError` | `http:GenericClientError` (`failed to parse: ...`) |
| `Listener.init` | `http:ListenerError` | `http:GenericListenerError`; a missing server certificate or key file reports `Key file or server certificates file not found` |
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
| With a `body` detail | The body, sent as a status code response body is; any `headers` detail is added |
| Without a `body` detail | An `http:ErrorPayload` JSON: `timestamp`, `status`, `reason`, `message`, `path` (with the query string), `method` |

Each returned error is logged as `unhandled error returned from the service`. A request that matches no service or resource answers 404, and one whose method matches no resource answers 405, with the same `http:ErrorPayload` body. An `OPTIONS` request also answers 405 when no `options` or `default` resource matches it.

