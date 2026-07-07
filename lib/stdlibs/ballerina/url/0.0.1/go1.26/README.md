# Ballerina URL Library

## Overview

This module provides the URL encoding/decoding functions.

URL encoding stands for encoding certain characters in a URL by replacing them with one or more character triplets that consist of the percent character `%` followed by two hexadecimal digits. The two hexadecimal digits of the triplet(s) represent the numeric value of the replaced character.

The Ballerina `url` module facilitates APIs to encode and decode a URL or part of a URL.

## Key Functionalities

- Encode a URL or part of a URL.
- Decode a URL or part of a URL.
- Support for different character encodings (UTF-8, ISO-8859-1, US-ASCII, UTF-16, UTF-16BE, UTF-16LE).

## Examples

```ballerina
import ballerina/io;
import ballerina/url;

public function main() {
    string value = "https://www.example.com/search?q=ballerina programming";
    string encodedUrl = check url:encode(value, "UTF-8");
    io:println("Encoded URL: ", encodedUrl);

    string decodedUrl = check url:decode(encodedUrl, "UTF-8");
    io:println("Decoded URL: ", decodedUrl);
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
| Encoding a URL | Supported | |
| Decoding a URL | Supported | |
| Support for different character encodings | Supported | |
| Specific error types | Not Yet Supported | Error handling is implemented but errors are returned as a generic error type. |

### Notable Behavioural Changes

There are **no** notable behavioural changes in the Go-native version compared to the original jBallerina implementation for the currently supported features.
