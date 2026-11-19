# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the `http` error type hierarchy, the first step in completing the `http` service and listener surface.

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
