# Ballerina MIME Library

## Overview

The `ballerina/mime` library provides utilities for working with MIME (Multipurpose Internet Mail Extensions) types and entities as defined by RFC 2045/2046. It covers media type parsing and construction, content disposition handling, entity header and body management (text, JSON, binary, and multipart), and Base64 encoding/decoding.

## Key Functionalities

- Parse and construct MIME media types (`MediaType`) including primary type, sub-type, suffix, and parameters.
- Parse and construct content disposition headers (`ContentDisposition`) including filename, name, and parameters.
- Manage MIME entity headers: set, get, add, remove, and check presence.
- Manage entity content metadata: content type, content ID, content length, and content disposition.
- Set and retrieve entity body payloads as text, JSON, or byte arrays, with every accessor able to convert from whatever the body was actually set as.
- Set and extract multipart (`multipart/form-data`, etc.) body parts as an `Entity[]`.
- Perform Base64 encoding and decoding of strings and byte arrays using MIME-compatible line folding.
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
| Entity text body | Supported | `setText`, `getText` — `getText()` also converts from a JSON or byte[] body, matching jBallerina's data-source model |
| Entity JSON body | Supported | `setJson`, `getJson` — `getJson()` also parses a text or byte[] body as JSON |
| Entity byte array body | Supported | `setByteArray`, `getByteArray` — `getByteArray()` also encodes a text or JSON body to bytes |
| Entity generic body dispatch | Supported | `setBody(string\|json\|byte[]\|Entity[])` |
| Entity multipart body | Partially Supported | `setBodyParts`, `getBodyParts` — flat (single-level) multipart only; a part whose own body is itself multipart is not recursively decoded. Per-part `Content-Type` defaults to `text/plain` when the wire omits it, matching jBallerina. `getBodyPartsAsChannel` is not implemented — it returns `io:ReadableByteChannel`, an unrelated, still-unimplemented io type |
| Base64 encoding and decoding | Supported | `base64Encode`, `base64Decode`, `base64EncodeBlob`, `base64DecodeBlob` |
| Entity XML body | Not Yet Supported | `setXml`, `getXml` require XML type support |
| Module-level error type | Partially Supported | `mime:Error` and all subtypes (`InvalidContentTypeError`, `ParserError`, `HeaderNotFoundError`, `EncodeError`, `DecodeError`, etc.) are plain `error` aliases; `distinct` type descriptor not yet supported |

### Notable Behavioural Changes

There are **no** notable behavioural changes in the Go-native version compared to the original jBallerina implementation for the currently supported features.
