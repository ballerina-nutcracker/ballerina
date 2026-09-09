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
| Entity text body | Supported | `setText`, `getText` — `getText()` also converts from a JSON or byte[] body, matching jBallerina's data-source model |
| Entity JSON body | Supported | `setJson`, `getJson` — `getJson()` also parses a text or byte[] body as JSON |
| Entity XML body | Supported | `setXml`, `getXml` — `getXml()` also parses a text or byte[] body as XML, matching jBallerina's data-source model |
| Entity byte array body | Supported | `setByteArray`, `getByteArray` — `getByteArray()` also encodes a text or JSON body to bytes |
| Entity body from a file | Supported | `setFileAsEntityBody(filePath, contentType)` — sets the file as a lazy data source, read on demand by whichever accessor materializes the body first, matching jBallerina. A failed file open panics, matching jBallerina's own `checkpanic io:openReadableFile` |
| Entity generic body dispatch | Supported | `setBody(string\|xml\|json\|byte[]\|Entity[])` |
| Entity multipart body | Supported | `setBodyParts`, `getBodyParts` — `message/*` bodies return a `ParserError`; jBallerina silently returns an empty array for these instead |
| Base64 encoding and decoding | Supported | `base64Encode`, `base64Decode`, `base64EncodeBlob`, `base64DecodeBlob` — `base64Encode`/`base64Decode` also accept an `io:ReadableByteChannel`, reading it fully and returning a freshly-constructed channel wrapping the result (charset is not applied to the channel form, matching jBallerina) |
| Module-level error type | Partially Supported | `mime:Error` and all subtypes (`InvalidContentTypeError`, `ParserError`, `HeaderNotFoundError`, `EncodeError`, `DecodeError`, etc.) are plain `error` aliases; `distinct` type descriptor not yet supported |

### Notable Behavioural Changes

- **A missing multipart boundary is a `ParserError`, not a silent empty array.** jBallerina's `getBodyParts()` only ever decodes from a byte-channel data source set internally by the HTTP transport for an inbound request/response; a manually-constructed `Entity` (via `setByteArray`) has a plain byte-array data source instead, so `getBodyParts()` silently returns an empty `Entity[]` rather than decoding or erroring — jBallerina has no public API that decodes an arbitrary `byte[]` as multipart at all. This port has one representation for an `Entity`'s body regardless of how it was constructed, so it always attempts to decode a composite (`multipart/*` or `message/*`) byte-array body and surfaces a missing boundary as a `ParserError` — the more debuggable choice, and unavoidable given this port lets a manually-constructed `Entity` decode multipart content at all (which jBallerina's public API cannot exercise).
