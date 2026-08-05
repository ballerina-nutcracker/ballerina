// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

// ── Errors ───────────────────────────────────────────────────────────────────

// Note: distinct error types are not yet supported; all subtypes are plain error aliases.

# The base error type for the `mime` module.
public type Error error;

# Represents an error that occurred while encoding data.
public type EncodeError error;

# Represents an error that occurred while decoding data.
public type DecodeError error;

# Represents a generic MIME-related error.
public type GenericMimeError error;

# Represents an error that occurred while setting a header.
public type SetHeaderError error;

# Represents an error due to an invalid header value.
public type InvalidHeaderValueError error;

# Represents an error due to an invalid header parameter.
public type InvalidHeaderParamError error;

# Represents an error due to an invalid content-length value.
public type InvalidContentLengthError error;

# Represents an error when a requested header cannot be found.
public type HeaderNotFoundError error;

# Represents an error due to an invalid header operation.
public type InvalidHeaderOperationError error;

# Represents an error that occurred during serialization.
public type SerializationError error;

# Represents an error that occurred while parsing an entity body.
public type ParserError error;

# Represents an error due to an invalid content-type value.
public type InvalidContentTypeError error;

# Represents an error when a header is unavailable.
public type HeaderUnavailableError error;

# Represents an error triggered when an idle timeout is reached.
public type IdleTimeoutTriggeredError error;

# Represents an error when no content is available.
public type NoContentError error;

// ── Media type constants ─────────────────────────────────────────────────────

# Media type for octet-stream.
public const APPLICATION_OCTET_STREAM = "application/octet-stream";

# Media type for JSON.
public const APPLICATION_JSON = "application/json";

# Media type for XML.
public const APPLICATION_XML = "application/xml";

# Media type for SVG+XML.
public const APPLICATION_SVG_XML = "application/svg+xml";

# Media type for XHTML+XML.
public const APPLICATION_XHTML_XML = "application/xhtml+xml";

# Media type for SOAP+XML.
public const APPLICATION_SOAP_XML = "application/soap+xml";

# Media type for URL-encoded form data.
public const APPLICATION_FORM_URLENCODED = "application/x-www-form-urlencoded";

# Media type for PDF.
public const APPLICATION_PDF = "application/pdf";

# Media type for JPEG images.
public const IMAGE_JPEG = "image/jpeg";

# Media type for GIF images.
public const IMAGE_GIF = "image/gif";

# Media type for PNG images.
public const IMAGE_PNG = "image/png";

# Media type for multipart form data.
public const MULTIPART_FORM_DATA = "multipart/form-data";

# Media type for multipart mixed content.
public const MULTIPART_MIXED = "multipart/mixed";

# Media type for multipart alternative content.
public const MULTIPART_ALTERNATIVE = "multipart/alternative";

# Media type for multipart related content.
public const MULTIPART_RELATED = "multipart/related";

# Media type for multipart parallel content.
public const MULTIPART_PARALLEL = "multipart/parallel";

# Media type for plain text.
public const TEXT_PLAIN = "text/plain";

# Media type for HTML.
public const TEXT_HTML = "text/html";

# Media type for XML text.
public const TEXT_XML = "text/xml";

# Media type for server-sent events.
public const TEXT_EVENT_STREAM = "text/event-stream";

// ── Header names and native bindings ──────────────────────────────────────────

# The `boundary` parameter name for multipart content types.
public const BOUNDARY = "boundary";

# The `start` parameter name for multipart content types.
public const START = "start";

# The `type` parameter name for multipart content types.
public const TYPE = "type";

# The `charset` parameter name for content types.
public const CHARSET = "charset";

# The default charset used when none is specified.
public const DEFAULT_CHARSET = "UTF-8";

# The `Content-Id` header name.
public const CONTENT_ID = "content-id";

# The `Content-Length` header name.
public const CONTENT_LENGTH = "content-length";

# The `Content-Type` header name.
public const CONTENT_TYPE = "content-type";

# The `Content-Disposition` header name.
public const CONTENT_DISPOSITION = "content-disposition";

# Represents the `Content-Disposition` header of an entity.
public class ContentDisposition {

    # The filename parameter of the content disposition.
    public string fileName = "";
    # The disposition type, e.g. `attachment`, `form-data`.
    public string disposition = "";
    # The `name` parameter of the content disposition.
    public string name = "";
    # Additional parameters of the content disposition.
    public map<string> parameters = {};

    # Converts this `ContentDisposition` to its wire representation.
    #
    # + return - the string representation of this content disposition
    public isolated function toString() returns string {
        return convertContentDispositionToString(self);
    }
}

isolated function convertContentDispositionToString(ContentDisposition contentDisposition) returns string = external;

# Represents the primary type, sub-type, suffix, and parameters of a MIME media type.
public class MediaType {

    # The primary type, e.g. `application`, `text`.
    public string primaryType = "";
    # The sub-type, e.g. `json`, `plain`.
    public string subType = "";
    # The suffix, e.g. `xml` in `application/svg+xml`.
    public string suffix = "";
    # Additional parameters of the media type, e.g. `charset`.
    public map<string> parameters = {};

    # Returns the base type, i.e. `primaryType/subType`, without any parameters.
    #
    # + return - the base media type string
    public isolated function getBaseType() returns string {
        return string `${self.primaryType}/${self.subType}`;
    }

    // Note: jBallerina uses the elvis operator (?:) which is not yet supported.
    // Rewritten using explicit if/else.
    # Converts this `MediaType` to its wire representation, including parameters.
    #
    # + return - the string representation of this media type
    public isolated function toString() returns string {
        string contentType = self.getBaseType();
        string[] keys = self.parameters.keys();
        if keys.length() > 0 {
            contentType = contentType + "; ";
        }
        foreach int i in 0 ..< keys.length() {
            string key = keys[i];
            string value = "";
            string? mapVal = self.parameters[key];
            if mapVal is string {
                value = mapVal;
            }
            if i > 0 {
                contentType = contentType + ";";
            }
            contentType = contentType + string `${key}=${value}`;
        }
        return contentType;
    }
}

# Represents a MIME entity: headers, content metadata, and a body (text, JSON,
# byte array, or multipart body parts).
public class Entity {

    private MediaType? cType;
    private string cId;
    private int cLength;
    private ContentDisposition? cDisposition;
    private map<string[]> headerMap;
    private string[] headerNames;

    public isolated function init() {
        self.cType = ();
        self.cId = "";
        self.cLength = 0;
        self.cDisposition = ();
        self.headerMap = {};
        self.headerNames = [];
    }

    # Sets the content type of this entity.
    #
    # + mediaType - the media type string to set
    # + return - an `InvalidContentTypeError` if `mediaType` cannot be parsed
    public isolated function setContentType(string mediaType) returns InvalidContentTypeError? {
        self.cType = check getMediaType(mediaType);
        self.setHeader(CONTENT_TYPE, mediaType);
        return;
    }

    # Returns the content type of this entity, or `""` if not set.
    #
    # + return - the content type string
    public isolated function getContentType() returns string {
        string contentTypeHeaderValue = "";
        string|HeaderNotFoundError value = self.getHeader(CONTENT_TYPE);
        if value is string {
            contentTypeHeaderValue = value;
        }
        return contentTypeHeaderValue;
    }

    # Sets the content ID of this entity.
    #
    # + contentId - the content ID to set
    public isolated function setContentId(string contentId) {
        self.cId = contentId;
        self.setHeader(CONTENT_ID, contentId);
    }

    # Returns the content ID of this entity, or `""` if not set.
    #
    # + return - the content ID string
    public isolated function getContentId() returns string {
        string contentId = "";
        string|HeaderNotFoundError value = self.getHeader(CONTENT_ID);
        if value is string {
            contentId = value;
        }
        return contentId;
    }

    # Sets the content length of this entity.
    #
    # + contentLength - the content length to set
    public isolated function setContentLength(int contentLength) {
        self.cLength = contentLength;
        self.setHeader(CONTENT_LENGTH, externIntToString(contentLength));
    }

    # Returns the content length of this entity, or `-1` if not set.
    #
    # + return - the content length, or an `error` if the header value isn't a valid integer
    public isolated function getContentLength() returns int|error {
        string contentLength = "";
        string|HeaderNotFoundError length = self.getHeader(CONTENT_LENGTH);
        if length is string {
            contentLength = length;
        }
        if contentLength == "" {
            return -1;
        }
        return externParseInt(contentLength);
    }

    # Sets the content disposition of this entity.
    #
    # + contentDisposition - the content disposition to set
    public isolated function setContentDisposition(ContentDisposition contentDisposition) {
        self.cDisposition = contentDisposition;
        self.setHeader(CONTENT_DISPOSITION, contentDisposition.toString());
    }

    # Returns the content disposition of this entity, or a default (empty) one if not set.
    #
    # + return - the content disposition
    public isolated function getContentDisposition() returns ContentDisposition {
        string contentDispositionVal = "";
        string|HeaderNotFoundError value = self.getHeader(CONTENT_DISPOSITION);
        if value is string {
            contentDispositionVal = value;
        }
        return getContentDispositionObject(contentDispositionVal);
    }

    # Sets the body of this entity, dispatching by the runtime type of `entityBody`.
    #
    # + entityBody - the body to set: text, JSON, byte array, or multipart body parts
    public isolated function setBody(string|json|byte[]|Entity[] entityBody) {
        if entityBody is string {
            self.setText(entityBody);
        } else if entityBody is byte[] {
            self.setByteArray(entityBody);
        } else if entityBody is Entity[] {
            self.setBodyParts(entityBody);
        } else if entityBody is json {
            self.setJson(entityBody);
        }
    }

    # Sets the body of this entity as JSON.
    #
    # + jsonContent - the JSON content to set
    # + contentType - the content type to set; defaults to `application/json`
    public isolated function setJson(json jsonContent, string contentType = "application/json") {
        externSetJson(self, jsonContent, contentType);
        self.setHeader(CONTENT_TYPE, contentType);
    }

    # Extracts the entity body as JSON, converting from a text or byte[] body if necessary.
    #
    # + return - the JSON body, or a `ParserError` if it cannot be parsed as JSON
    public isolated function getJson() returns json|ParserError {
        return externGetJson(self);
    }

    # Sets the body of this entity as text.
    #
    # + textContent - the text content to set
    # + contentType - the content type to set; defaults to `text/plain`
    public isolated function setText(string textContent, string contentType = "text/plain") {
        externSetText(self, textContent, contentType);
        self.setHeader(CONTENT_TYPE, contentType);
    }

    # Extracts the entity body as text, converting from a JSON or byte[] body if necessary.
    #
    # + return - the text body, or a `ParserError` if it cannot be read as text
    public isolated function getText() returns string|ParserError {
        return externGetText(self);
    }

    # Sets the body of this entity as a byte array.
    #
    # + blobContent - the byte array content to set
    # + contentType - the content type to set; defaults to `application/octet-stream`
    public isolated function setByteArray(byte[] blobContent, string contentType = "application/octet-stream") {
        externSetByteArray(self, blobContent, contentType);
        self.setHeader(CONTENT_TYPE, contentType);
    }

    # Extracts the entity body as a byte array, converting from a text or JSON body if necessary.
    #
    # + return - the byte array body, or a `ParserError` if it cannot be encoded to bytes
    public isolated function getByteArray() returns byte[]|ParserError {
        return externGetByteArray(self);
    }

    # Sets the body parts of the entity, marking it as a multipart entity.
    #
    # + bodyParts - The body parts to be set
    # + contentType - Optional MIME type; defaults to `multipart/form-data`
    public isolated function setBodyParts(Entity[] bodyParts, string contentType = MULTIPART_FORM_DATA) {
        externSetBodyParts(self, bodyParts, contentType);
        self.setHeader(CONTENT_TYPE, contentType);
    }

    # Extracts body parts from a multipart entity.
    #
    # + return - An array of body parts, or a `ParserError` if the entity is not a composite
    # (`multipart/*` or `message/*`) media type, or the body cannot be decoded
    public isolated function getBodyParts() returns Entity[]|ParserError {
        return externGetBodyParts(self);
    }

    # Returns the first value of the given header.
    #
    # + headerName - the header name, case-insensitive
    # + return - the header value, or a `HeaderNotFoundError` if it isn't set
    public isolated function getHeader(string headerName) returns string|HeaderNotFoundError {
        string[] value = check self.getHeaders(headerName.toLowerAscii());
        return value[0];
    }

    # Returns all values of the given header.
    #
    # + headerName - the header name, case-insensitive
    # + return - the header values, or a `HeaderNotFoundError` if it isn't set
    public isolated function getHeaders(string headerName) returns string[]|HeaderNotFoundError {
        string lowerCaseHeaderName = headerName.toLowerAscii();
        string[]? value = self.headerMap[lowerCaseHeaderName];
        if value is () {
            return error HeaderNotFoundError("Http header does not exist");
        }
        return [...value];
    }

    # Returns the names of all headers set on this entity, in their original casing.
    #
    # + return - the header names
    public isolated function getHeaderNames() returns string[] {
        return [...self.headerNames];
    }

    # Adds a value to the given header, appending to any existing values.
    #
    # + headerName - the header name, case-insensitive
    # + headerValue - the value to add
    public isolated function addHeader(string headerName, string headerValue) {
        string lowerCaseHeaderName = headerName.toLowerAscii();
        string[]|HeaderNotFoundError headerList = self.getHeaders(lowerCaseHeaderName);
        if headerList is string[] {
            headerList.push(headerValue);
            self.headerMap[lowerCaseHeaderName] = headerList;
        } else {
            self.setHeader(headerName, headerValue);
        }
    }

    # Sets the given header, replacing any existing value(s).
    #
    # + headerName - the header name, case-insensitive
    # + headerValue - the value to set
    public isolated function setHeader(string headerName, string headerValue) {
        string[] value = [headerValue];
        self.headerMap[headerName.toLowerAscii()] = value;
        string? caseSensitiveValue = getCaseSensitiveHeaderName(self.headerNames, headerName);
        if caseSensitiveValue is () {
            self.headerNames.push(headerName);
        }
    }

    # Removes the given header, if present.
    #
    # + headerName - the header name, case-insensitive
    public isolated function removeHeader(string headerName) {
        string lowerName = headerName.toLowerAscii();
        if !self.headerMap.hasKey(lowerName) {
            return;
        }
        _ = self.headerMap.remove(lowerName);
        self.headerNames = from string name in self.headerNames
            where name.toLowerAscii() != lowerName
            select name;
    }

    # Removes all headers set on this entity.
    public isolated function removeAllHeaders() {
        self.headerMap = {};
        self.headerNames = [];
    }

    # Checks whether the given header is set on this entity.
    #
    # + headerName - the header name, case-insensitive
    # + return - `true` if the header is set
    public isolated function hasHeader(string headerName) returns boolean {
        return self.headerMap.hasKey(headerName.toLowerAscii());
    }
}

isolated function externSetJson(Entity entity, json jsonContent, string contentType) = external;

isolated function externGetJson(Entity entity) returns json|ParserError = external;

isolated function externSetText(Entity entity, string textContent, string contentType) = external;

isolated function externGetText(Entity entity) returns string|ParserError = external;

isolated function externSetByteArray(Entity entity, byte[] byteArray, string contentType) = external;

isolated function externGetByteArray(Entity entity) returns byte[]|ParserError = external;

isolated function externSetBodyParts(Entity entity, Entity[] bodyParts, string contentType) = external;

isolated function externGetBodyParts(Entity entity) returns Entity[]|ParserError = external;

isolated function externParseInt(string s) returns int|error = external;

isolated function externIntToString(int n) returns string = external;

isolated function getCaseSensitiveHeaderName(string[] headerNames, string headerName) returns string? {
    foreach string name in headerNames {
        if name.toLowerAscii() == headerName.toLowerAscii() {
            return name;
        }
    }
    return;
}

# Parses a `Content-Type` header value into a `MediaType`.
#
# + contentType - the content type string to parse
# + return - the parsed media type, or an `InvalidContentTypeError` if it cannot be parsed
public isolated function getMediaType(string contentType) returns MediaType|InvalidContentTypeError = external;

# Parses a `Content-Disposition` header value into a `ContentDisposition`.
#
# + contentDisposition - the content disposition string to parse
# + return - the parsed content disposition
public isolated function getContentDispositionObject(string contentDisposition) returns ContentDisposition = external;

# Encodes a string or byte array using MIME-compatible Base64 encoding.
#
# + contentToBeEncoded - the string or byte array to encode
# + charset - the charset to use when the input is a string; defaults to `utf-8`
# + return - the encoded value, in the same shape as the input, or an `EncodeError`
public isolated function base64Encode((string|byte[]) contentToBeEncoded, string charset = "utf-8")
        returns (string|byte[]|EncodeError) = external;

# Decodes a Base64-encoded string or byte array.
#
# + contentToBeDecoded - the string or byte array to decode
# + charset - the charset to use when the input is a string; defaults to `utf-8`
# + return - the decoded value, in the same shape as the input, or a `DecodeError`
public isolated function base64Decode((string|byte[]) contentToBeDecoded, string charset = "utf-8")
        returns (string|byte[]|DecodeError) = external;

# Encodes a byte array using MIME-compatible Base64 encoding.
#
# + valueToBeEncoded - the byte array to encode
# + return - the encoded byte array, or an `EncodeError`
public isolated function base64EncodeBlob(byte[] valueToBeEncoded) returns byte[]|EncodeError {
    string|byte[]|EncodeError result = base64Encode(valueToBeEncoded);
    if result is byte[]|EncodeError {
        return result;
    }
    return error EncodeError("Error occurred while encoding byte[]");
}

# Decodes a Base64-encoded byte array.
#
# + valueToBeDecoded - the byte array to decode
# + return - the decoded byte array, or a `DecodeError`
public isolated function base64DecodeBlob(byte[] valueToBeDecoded) returns byte[]|DecodeError {
    string|byte[]|DecodeError result = base64Decode(valueToBeDecoded);
    if result is byte[]|DecodeError {
        return result;
    }
    return error DecodeError("Error occurred while decoding byte[]");
}
