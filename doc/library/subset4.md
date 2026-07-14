# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with multipart
(`multipart/form-data`, etc.) body support in `mime`, plus the `http`
`Request`/`Response` methods that build on it.

## [mime](https://github.com/ballerina-platform/module-ballerina-mime/blob/master/docs/spec/spec.md)

| Feature | Notes |
|---|---|
| `Entity.setBodyParts` / `getBodyParts` | Sets/extracts multipart body parts as an `Entity[]`. Flat (single-level) multipart only — a part whose own body is itself multipart is not recursively decoded. A part with no `Content-Type` header defaults to `text/plain`, matching jBallerina's underlying MIME library. |
| `Entity.getText` / `getJson` / `getByteArray` cross-conversion | Every accessor now lazily converts from whatever the body was actually set as (matching jBallerina's data-source model) instead of requiring an exact kind match — e.g. `getByteArray()` on a text-body entity succeeds instead of erroring. |

`getBodyPartsAsChannel` is not implemented — it returns `io:ReadableByteChannel`,
a separate, still-unimplemented io type unrelated to multipart parsing itself.

## [http](https://github.com/ballerina-platform/module-ballerina-http/blob/master/docs/spec/spec.md)

| Feature | Notes |
|---|---|
| `Request`/`Response` `setBodyParts` / `getBodyParts` | Multipart request/response bodies, built on `mime`'s new `Entity[]` support. `setBodyParts`'s optional `contentType` follows jBallerina's exact precedence: an explicit override wins, otherwise the request/response's existing `Content-Type` is kept if already set, otherwise `multipart/form-data` is used; a boundary is generated if the resolved content type doesn't already carry one. |
| `setPayload` | Widened to accept `mime:Entity[]` in addition to `json`/`byte[]`, dispatching to `setBodyParts`. |
| `RequestMessage` | Widened to `json\|Request\|mime:Entity[]` — the client's `post`/`put`/`patch`/`delete`/`execute` methods now accept a multipart body directly. |
