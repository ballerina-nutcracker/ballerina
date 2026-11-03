# Ballerina MIME Library

## Overview

The `ballerina/mime` library provides utilities for working with MIME (Multipurpose Internet Mail Extensions) types and entities as defined by RFC 2045/2046. It covers media type parsing and construction, content disposition handling, entity header and body management (text, XML, JSON, binary, and multipart), and Base64 encoding/decoding.

## Key Functionalities

- Parse and construct MIME media types (`MediaType`) including primary type, sub-type, suffix, and parameters.
- Parse and construct content disposition headers (`ContentDisposition`) including filename, name, and parameters.
- Manage MIME entity headers: set, get, add, remove, and check presence.
- Manage entity content metadata: content type, content ID, content length, and content disposition.
- Set and retrieve entity body payloads as text, XML, JSON, or byte arrays, with every accessor able to convert from whatever the body was actually set as. Set a body directly from a file.
- Set and extract multipart (`multipart/form-data`, etc.) body parts as an `Entity[]`.
- Perform Base64 encoding and decoding of strings, byte arrays, and byte channels using MIME-compatible line folding.
- Predefined constants for common media type strings and header names.

## Examples

```ballerina
import ballerina/io;
import ballerina/mime;

public function main() returns error? {
    mime:MediaType mt = check mime:getMediaType("application/json; charset=UTF-8");
    io:println(mt.primaryType);          // application
    io:println(mt.getBaseType());        // application/json
    io:println(mt.parameters["charset"]); // UTF-8

    mime:Entity entity = new ();
    entity.setHeader("Content-Type", mime:APPLICATION_JSON);
    io:println(entity.hasHeader("content-type")); // true
    entity.setText("Hello");
    string|mime:ParserError text = entity.getText();
    if text is string {
        io:println(text); // Hello
    }

    string|byte[]|mime:EncodeError encoded = mime:base64Encode("Hello");
    if encoded is string {
        io:println(encoded); // SGVsbG8=
    }
}
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Media type MIME string constants | Supported | `APPLICATION_JSON`, `TEXT_PLAIN`, `APPLICATION_OCTET_STREAM`, `MULTIPART_FORM_DATA`, `TEXT_HTML`, `IMAGE_JPEG`, and 14 others |
| Header and charset constants | Supported | `CONTENT_TYPE`, `CONTENT_LENGTH`, `CONTENT_ID`, `CONTENT_DISPOSITION`, `DEFAULT_CHARSET`, `CHARSET`, `BOUNDARY`, `START`, `TYPE` |
| MediaType class | Supported | Fields `primaryType`, `subType`, `suffix`, `parameters`; methods `getBaseType()` and `toString()` |
| ContentDisposition class | Supported | Fields `fileName`, `disposition`, `name`, `parameters`; method `toString()` |
| Media type parsing | Supported | `getMediaType(contentType)` — returns `MediaType\|InvalidContentTypeError` |
| Content disposition parsing | Supported | `getContentDispositionObject(contentDisposition)` |
| Entity header management | Supported | `setHeader`, `getHeader`, `getHeaders`, `getHeaderNames`, `addHeader`, `removeHeader`, `removeAllHeaders`, `hasHeader` |
| Entity content metadata | Supported | `setContentType`, `getContentType`, `setContentId`, `getContentId`, `setContentLength`, `getContentLength`, `setContentDisposition`, `getContentDisposition` |
| Entity text body | Supported | `setText`, `getText` — `getText()` also converts from a JSON or byte[] body, matching jBallerina's data-source model; a `setByteArray` body is always decoded as UTF-8 (the `charset` param is ignored, as in jBallerina), while a file-backed body or a part returned by `getBodyParts()` is decoded in the Content-Type `charset` (UTF-8 if absent), as in jBallerina; each malformed UTF-8 sequence is replaced by U+FFFD; an unknown charset returns a `ParserError` whose message ends with the charset name, as in jBallerina |
| Entity JSON body | Supported | `setJson`, `getJson` — `getJson()` also parses a text or byte[] body as JSON |
| Entity XML body | Supported | `setXml`, `getXml` — `getXml()` also parses a text or byte[] body as XML, matching jBallerina's data-source model |
| Entity byte array body | Supported | `setByteArray`, `getByteArray` — `getByteArray()` also encodes a text, JSON or XML body to bytes in the Content-Type `charset` (UTF-8 if absent), writing `?` for a character the charset cannot represent and returning a `ParserError` for an unknown charset, matching jBallerina; Java's charset aliases such as `ascii` and `iso8859-1` resolve to US-ASCII and ISO-8859-1 as in Java |
| Entity body from a file | Supported | `setFileAsEntityBody(filePath, contentType)` — sets the file as a lazy data source, read on demand by whichever accessor materializes the body first, matching jBallerina. A failed file open panics, matching jBallerina's own `checkpanic io:openReadableFile` |
| Entity generic body dispatch | Partially Supported | `setBody(string\|xml\|json\|byte[]\|Entity[])`; the `stream<byte[], io:Error?>` arm is not yet supported |
| Entity multipart body | Supported | `setBodyParts`, `getBodyParts` — a byte[] or file-backed body is decoded, and a part without a `Content-Type` header is left without one, matching jBallerina; an entity with no body, or with a text, JSON or XML body, returns an empty array, matching jBallerina. `message/*` bodies return a `ParserError`; jBallerina silently returns an empty array for these instead. When body parts are serialized (by `ballerina/http`), a part whose own body is body parts is encoded recursively as a nested multipart body, with a generated boundary added to that part's Content-Type |
| Entity byte stream body | Not Yet Supported | `getByteStream`, `setByteStream` — no stream-backed entity body yet |
| Multipart body parts as a stream | Not Yet Supported | `getBodyPartsAsStream` |
| Base64 encoding and decoding | Supported | `base64Encode`, `base64Decode`, `base64EncodeBlob`, `base64DecodeBlob` — `base64Encode`/`base64Decode` also accept an `io:ReadableByteChannel`, reading it fully and returning a freshly-constructed channel wrapping the result (charset is not applied to the channel form, matching jBallerina). Characters outside the base64 alphabet are skipped when decoding, and a character the charset cannot represent is encoded as `?`, matching Java's MIME base64 codec |
| Module-level error type | Partially Supported | `mime:Error` and all subtypes (`InvalidContentTypeError`, `ParserError`, `HeaderNotFoundError`, `EncodeError`, `DecodeError`, etc.) are plain `error` aliases; `distinct` type descriptor not yet supported |

### Notable Behavioural Changes

- **A `byte[]` multipart body is decoded.** jBallerina's `getBodyParts()` decodes only a byte-channel body (an inbound HTTP body or `setFileAsEntityBody`) and returns an empty `Entity[]` for a `setByteArray` body; the Go-native version decodes a `byte[]` body too, and reports a missing boundary, or a non-empty body holding no part, as a `ParserError`.
- **Invalid content type messages are not wrapped.** jBallerina's `getMediaType()` puts the string form of an inner error into the message, e.g. `error InvalidContentTypeError ("Unable to find a sub type.")`; the Go-native version returns the javax.activation parse message itself, e.g. `Unable to find a sub type.`.
- **Content disposition parameter names are case-insensitive.** jBallerina only recognises lower-case `name` and `filename`, so `FORM-DATA; NAME="n"; FileName="f"` leaves both quoted in `parameters`, and `toString()` throws a Java `NullPointerException` for some unquoted values or values containing `;`; the Go-native version matches `name` and `filename` case-insensitively (as RFC 2183 specifies), unquotes them into the `name` and `fileName` fields, and never fails in `toString()`.
- **A scalar JSON body can be read with `getJson()`.** jBallerina fails with a `ParserError` wrapping a Java `ClassCastException` when the body is a JSON scalar such as `12`; the Go-native version returns the scalar value.
- **The byte form of a JSON body is always valid JSON.** jBallerina's `getByteArray()` on a `setJson` body writes string contents unescaped (so `"` or a newline inside a string yields invalid JSON) and panics with a `TypeCastError` when the body is a top-level JSON array; the Go-native version escapes strings the same way `getText()` does and serializes arrays.
- **Parameter and header order is deterministic but not jBallerina's.** `MediaType.parameters` (and therefore `MediaType.toString()`) keeps the order the parameters appear in the source string; jBallerina's order follows Java `Hashtable` hashing. Headers of a decoded multipart part are added in sorted name order, since Go's multipart reader does not keep their wire order.
