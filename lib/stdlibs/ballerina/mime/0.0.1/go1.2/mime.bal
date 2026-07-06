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

// ── errors.bal ──────────────────────────────────────────────────────────────

// Note: distinct error types are not yet supported; all subtypes are plain error aliases.
public type Error error;

public type EncodeError error;

public type DecodeError error;

public type GenericMimeError error;

public type SetHeaderError error;

public type InvalidHeaderValueError error;

public type InvalidHeaderParamError error;

public type InvalidContentLengthError error;

public type HeaderNotFoundError error;

public type InvalidHeaderOperationError error;

public type SerializationError error;

public type ParserError error;

public type InvalidContentTypeError error;

public type HeaderUnavailableError error;

public type IdleTimeoutTriggeredError error;

public type NoContentError error;

// ── media_types.bal ─────────────────────────────────────────────────────────

public const string APPLICATION_OCTET_STREAM = "application/octet-stream";

public const string APPLICATION_JSON = "application/json";

public const string APPLICATION_XML = "application/xml";

public const string APPLICATION_SVG_XML = "application/svg+xml";

public const string APPLICATION_XHTML_XML = "application/xhtml+xml";

public const string APPLICATION_SOAP_XML = "application/soap+xml";

public const string APPLICATION_FORM_URLENCODED = "application/x-www-form-urlencoded";

public const string APPLICATION_PDF = "application/pdf";

public const string IMAGE_JPEG = "image/jpeg";

public const string IMAGE_GIF = "image/gif";

public const string IMAGE_PNG = "image/png";

public const string MULTIPART_FORM_DATA = "multipart/form-data";

public const string MULTIPART_MIXED = "multipart/mixed";

public const string MULTIPART_ALTERNATIVE = "multipart/alternative";

public const string MULTIPART_RELATED = "multipart/related";

public const string MULTIPART_PARALLEL = "multipart/parallel";

public const string TEXT_PLAIN = "text/plain";

public const string TEXT_HTML = "text/html";

public const string TEXT_XML = "text/xml";

public const string TEXT_EVENT_STREAM = "text/event-stream";

// ── natives.bal ─────────────────────────────────────────────────────────────

public const string BOUNDARY = "boundary";

public const string START = "start";

public const string TYPE = "type";

public const string CHARSET = "charset";

public const string DEFAULT_CHARSET = "UTF-8";

public const string CONTENT_ID = "content-id";

public const string CONTENT_LENGTH = "content-length";

public const string CONTENT_TYPE = "content-type";

public const string CONTENT_DISPOSITION = "content-disposition";

public class ContentDisposition {

    public string fileName = "";
    public string disposition = "";
    public string name = "";
    public map<string> parameters = {};

    public isolated function toString() returns string {
        return convertContentDispositionToString(self);
    }
}

isolated function convertContentDispositionToString(ContentDisposition contentDisposition) returns string = external;

public class MediaType {

    public string primaryType = "";
    public string subType = "";
    public string suffix = "";
    public map<string> parameters = {};

    public isolated function getBaseType() returns string {
        return self.primaryType + "/" + self.subType;
    }

    // Note: jBallerina uses the elvis operator (?:) which is not yet supported.
    // Rewritten using explicit if/else.
    public isolated function toString() returns string {
        string contentType = self.getBaseType();
        string[] arrKeys = self.parameters.keys();
        int size = arrKeys.length();
        if (size > 0) {
            contentType = contentType + "; ";
        }
        int index = 0;
        while (index < size) {
            string? mapVal = self.parameters[arrKeys[index]];
            string value = "";
            if mapVal is string {
                value = mapVal;
            }
            if (index == size - 1) {
                contentType = contentType + arrKeys[index] + "=" + value;
                break;
            } else {
                contentType = contentType + arrKeys[index] + "=" + value + ";";
                index = index + 1;
            }
        }
        return contentType;
    }
}

public class Entity {

    private MediaType? cType;
    private string cId;
    private int cLength;
    private ContentDisposition? cDisposition;
    private map<string[]> headerMap;
    private string[] headerNames;

    public function init() {
        self.cType = ();
        self.cId = "";
        self.cLength = 0;
        self.cDisposition = ();
        self.headerMap = {};
        self.headerNames = [];
    }

    public isolated function setContentType(string mediaType) returns InvalidContentTypeError? {
        self.cType = check getMediaType(mediaType);
        self.setHeader(CONTENT_TYPE, mediaType);
        return;
    }

    public isolated function getContentType() returns string {
        string contentTypeHeaderValue = "";
        var value = self.getHeader(CONTENT_TYPE);
        if (value is string) {
            contentTypeHeaderValue = value;
        }
        return contentTypeHeaderValue;
    }

    public isolated function setContentId(string contentId) {
        self.cId = contentId;
        self.setHeader(CONTENT_ID, contentId);
    }

    public isolated function getContentId() returns string {
        string contentId = "";
        var value = self.getHeader(CONTENT_ID);
        if (value is string) {
            contentId = value;
        }
        return contentId;
    }

    public isolated function setContentLength(int contentLength) {
        self.cLength = contentLength;
        self.setHeader(CONTENT_LENGTH, externIntToString(contentLength));
    }

    public isolated function getContentLength() returns int|error {
        string contentLength = "";
        var length = self.getHeader(CONTENT_LENGTH);
        if (length is string) {
            contentLength = length;
        }
        if (contentLength == "") {
            return -1;
        } else {
            return externParseInt(contentLength);
        }
    }

    public isolated function setContentDisposition(ContentDisposition contentDisposition) {
        self.cDisposition = contentDisposition;
        self.setHeader(CONTENT_DISPOSITION, contentDisposition.toString());
    }

    public isolated function getContentDisposition() returns ContentDisposition {
        string contentDispositionVal = "";
        var value = self.getHeader(CONTENT_DISPOSITION);
        if (value is string) {
            contentDispositionVal = value;
        }
        return getContentDispositionObject(contentDispositionVal);
    }

    public isolated function setBody(string|json|byte[] entityBody) {
        if (entityBody is string) {
            self.setText(entityBody);
        } else if (entityBody is byte[]) {
            self.setByteArray(entityBody);
        } else {
            self.setJson(entityBody);
        }
    }

    public isolated function setJson(json jsonContent, string contentType = "application/json") {
        externSetJson(self, jsonContent, contentType);
        self.setHeader(CONTENT_TYPE, contentType);
    }

    public isolated function getJson() returns json|ParserError {
        return externGetJson(self);
    }

    public isolated function setText(string textContent, string contentType = "text/plain") {
        externSetText(self, textContent, contentType);
        self.setHeader(CONTENT_TYPE, contentType);
    }

    public isolated function getText() returns string|ParserError {
        return externGetText(self);
    }

    public isolated function setByteArray(byte[] blobContent, string contentType = "application/octet-stream") {
        externSetByteArray(self, blobContent, contentType);
        self.setHeader(CONTENT_TYPE, contentType);
    }

    public isolated function getByteArray() returns byte[]|ParserError {
        return externGetByteArray(self);
    }

    public isolated function getHeader(string headerName) returns string|HeaderNotFoundError {
        string[]|HeaderNotFoundError value = self.getHeaders(headerName.toLowerAscii());
        if (value is string[]) {
            return value[0];
        } else {
            return value;
        }
    }

    public isolated function getHeaders(string headerName) returns string[]|HeaderNotFoundError {
        string lowerCaseHeaderName = headerName.toLowerAscii();
        string[]? value = self.headerMap[lowerCaseHeaderName];
        if (value is ()) {
            return error HeaderNotFoundError("Http header does not exist");
        } else {
            return value;
        }
    }

    public isolated function getHeaderNames() returns string[] {
        string[] result = [];
        foreach string name in self.headerNames {
            result.push(name);
        }
        return result;
    }

    public isolated function addHeader(string headerName, string headerValue) {
        string lowerCaseHeaderName = headerName.toLowerAscii();
        var headerList = self.getHeaders(lowerCaseHeaderName);
        if (headerList is string[]) {
            headerList.push(headerValue);
        } else {
            self.setHeader(headerName, headerValue);
        }
    }

    public isolated function setHeader(string headerName, string headerValue) {
        string[] value = [headerValue];
        self.headerMap[headerName.toLowerAscii()] = value;
        string? caseSensitiveValue = getCaseSensitiveHeaderName(self.headerNames, headerName);
        if caseSensitiveValue is () {
            self.headerNames.push(headerName);
        }
    }

    public isolated function removeHeader(string headerName) {
        if !(self.hasHeader(headerName)) {
            return;
        }
        _ = self.headerMap.remove(headerName.toLowerAscii());
        string lowerName = headerName.toLowerAscii();
        string[] newNames = [];
        foreach string name in self.headerNames {
            if !(name.toLowerAscii() == lowerName) {
                newNames.push(name);
            }
        }
        self.headerNames = newNames;
    }

    public isolated function removeAllHeaders() {
        self.headerMap = {};
        self.headerNames = [];
    }

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

isolated function externParseInt(string s) returns int|error = external;

isolated function externIntToString(int n) returns string = external;

isolated function getCaseSensitiveHeaderName(string[] headerNames, string headerName) returns string? {
    foreach string name in headerNames {
        if (name.toLowerAscii() == headerName.toLowerAscii()) {
            return name;
        }
    }
    return;
}

public isolated function getMediaType(string contentType) returns MediaType|InvalidContentTypeError = external;

public isolated function getContentDispositionObject(string contentDisposition) returns ContentDisposition = external;

public isolated function base64Encode((string|byte[]) contentToBeEncoded, string charset = "utf-8")
        returns (string|byte[]|EncodeError) = external;

public isolated function base64Decode((string|byte[]) contentToBeDecoded, string charset = "utf-8")
        returns (string|byte[]|DecodeError) = external;

public isolated function base64EncodeBlob(byte[] valueToBeEncoded) returns byte[]|EncodeError {
    var result = base64Encode(valueToBeEncoded);
    if (result is byte[]|EncodeError) {
        return result;
    } else {
        return error EncodeError("Error occurred while encoding byte[]");
    }
}

public isolated function base64DecodeBlob(byte[] valueToBeDecoded) returns byte[]|DecodeError {
    var result = base64Decode(valueToBeDecoded);
    if (result is byte[]|DecodeError) {
        return result;
    } else {
        return error DecodeError("Error occurred while decoding byte[]");
    }
}
