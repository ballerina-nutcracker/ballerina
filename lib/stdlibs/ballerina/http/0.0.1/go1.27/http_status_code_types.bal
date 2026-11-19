// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
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


# Defines the possible status code response record types.
public type StatusCodeResponse Continue|SwitchingProtocols|Processing|EarlyHints|Ok|Created|Accepted|
    NonAuthoritativeInformation|NoContent|ResetContent|PartialContent|MultiStatus|AlreadyReported|IMUsed|
    MultipleChoices|MovedPermanently|Found|SeeOther|NotModified|UseProxy|TemporaryRedirect|PermanentRedirect|
    BadRequest|Unauthorized|PaymentRequired|Forbidden|NotFound|MethodNotAllowed|NotAcceptable|
    ProxyAuthenticationRequired|RequestTimeout|Conflict|Gone|LengthRequired|PreconditionFailed|PayloadTooLarge|
    UriTooLong|UnsupportedMediaType|RangeNotSatisfiable|ExpectationFailed|MisdirectedRequest|UnprocessableEntity|
    Locked|FailedDependency|TooEarly|PreconditionRequired|UnavailableDueToLegalReasons|UpgradeRequired|TooManyRequests|
    RequestHeaderFieldsTooLarge|InternalServerError|NotImplemented|BadGateway|ServiceUnavailable|GatewayTimeout|
    HttpVersionNotSupported|VariantAlsoNegotiates|InsufficientStorage|LoopDetected|NotExtended|
    NetworkAuthenticationRequired|DefaultStatusCodeResponse;


# The `Status` object creates the distinction for the different response status code types.
#
# + code - The response status code
public type Status distinct object {
    public int code;
};

# The common attributed of response status code record type.
#
# + mediaType - The value of response `Content-type` header
# + headers - The response headers
# + body - The response payload
public type CommonResponse record {|
    string mediaType?;
    map<string|int|boolean|string[]|int[]|boolean[]> headers?;
    anydata body?;
|};

// Status code class declarations

# The default status code class.
public readonly distinct class DefaultStatus {
    *Status;

    public isolated function init(int code) {
        self.code = code;
    }
}

# Represents the status code of `STATUS_CONTINUE`.
#
# + code - The response status code
public readonly class StatusContinue {
    *Status;
    public STATUS_CONTINUE code = STATUS_CONTINUE;
    public isolated function init() {}
}

# Represents the status code of `STATUS_SWITCHING_PROTOCOLS`.
#
# + code - The response status code
public readonly class StatusSwitchingProtocols {
    *Status;
    public STATUS_SWITCHING_PROTOCOLS code = STATUS_SWITCHING_PROTOCOLS;
    public isolated function init() {}
}

# Represents the status code of `STATUS_PROCESSING`.
#
# + code - The response status code
public readonly class StatusProcessing {
    *Status;
    public STATUS_PROCESSING code = STATUS_PROCESSING;
    public isolated function init() {}
}

# Represents the status code of `STATUS_EARLY_HINTS`.
#
# + code - The response status code
public readonly class StatusEarlyHints {
    *Status;
    public STATUS_EARLY_HINTS code = STATUS_EARLY_HINTS;
    public isolated function init() {}
}

# Represents the status code of `STATUS_OK`.
#
# + code - The response status code
public readonly class StatusOK {
    *Status;
    public STATUS_OK code = STATUS_OK;
    public isolated function init() {}
}

# Represents the status code of `STATUS_CREATED`.
#
# + code - The response status code
public readonly class StatusCreated {
    *Status;
    public STATUS_CREATED code = STATUS_CREATED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_ACCEPTED`.
#
# + code - The response status code
public readonly class StatusAccepted {
    *Status;
    public STATUS_ACCEPTED code = STATUS_ACCEPTED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_NON_AUTHORITATIVE_INFORMATION`.
#
# + code - The response status code
public readonly class StatusNonAuthoritativeInformation {
    *Status;
    public STATUS_NON_AUTHORITATIVE_INFORMATION code = STATUS_NON_AUTHORITATIVE_INFORMATION;
    public isolated function init() {}
}

# Represents the status code of `STATUS_NO_CONTENT`.
#
# + code - The response status code
public readonly class StatusNoContent {
    *Status;
    public STATUS_NO_CONTENT code = STATUS_NO_CONTENT;
    public isolated function init() {}
}

# Represents the status code of `STATUS_RESET_CONTENT`.
#
# + code - The response status code
public readonly class StatusResetContent {
    *Status;
    public STATUS_RESET_CONTENT code = STATUS_RESET_CONTENT;
    public isolated function init() {}
}

# Represents the status code of `STATUS_PARTIAL_CONTENT`.
#
# + code - The response status code
public readonly class StatusPartialContent {
    *Status;
    public STATUS_PARTIAL_CONTENT code = STATUS_PARTIAL_CONTENT;
    public isolated function init() {}
}

# Represents the status code of `STATUS_MULTI_STATUS`.
#
# + code - The response status code
public readonly class StatusMultiStatus {
    *Status;
    public STATUS_MULTI_STATUS code = STATUS_MULTI_STATUS;
    public isolated function init() {}
}

# Represents the status code of `STATUS_ALREADY_REPORTED`.
#
# + code - The response status code
public readonly class StatusAlreadyReported {
    *Status;
    public STATUS_ALREADY_REPORTED code = STATUS_ALREADY_REPORTED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_IM_USED`.
#
# + code - The response status code
public readonly class StatusIMUsed {
    *Status;
    public STATUS_IM_USED code = STATUS_IM_USED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_MULTIPLE_CHOICES`.
#
# + code - The response status code
public readonly class StatusMultipleChoices {
    *Status;
    public STATUS_MULTIPLE_CHOICES code = STATUS_MULTIPLE_CHOICES;
    public isolated function init() {}
}

# Represents the status code of `STATUS_MOVED_PERMANENTLY`.
#
# + code - The response status code
public readonly class StatusMovedPermanently {
    *Status;
    public STATUS_MOVED_PERMANENTLY code = STATUS_MOVED_PERMANENTLY;
    public isolated function init() {}
}

# Represents the status code of `STATUS_FOUND`.
#
# + code - The response status code
public readonly class StatusFound {
    *Status;
    public STATUS_FOUND code = STATUS_FOUND;
    public isolated function init() {}
}

# Represents the status code of `STATUS_SEE_OTHER`.
#
# + code - The response status code
public readonly class StatusSeeOther {
    *Status;
    public STATUS_SEE_OTHER code = STATUS_SEE_OTHER;
    public isolated function init() {}
}

# Represents the status code of `STATUS_NOT_MODIFIED`.
#
# + code - The response status code
public readonly class StatusNotModified {
    *Status;
    public STATUS_NOT_MODIFIED code = STATUS_NOT_MODIFIED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_USE_PROXY`.
#
# + code - The response status code
public readonly class StatusUseProxy {
    *Status;
    public STATUS_USE_PROXY code = STATUS_USE_PROXY;
    public isolated function init() {}
}

# Represents the status code of `STATUS_TEMPORARY_REDIRECT`.
#
# + code - The response status code
public readonly class StatusTemporaryRedirect {
    *Status;
    public STATUS_TEMPORARY_REDIRECT code = STATUS_TEMPORARY_REDIRECT;
    public isolated function init() {}
}

# Represents the status code of `STATUS_PERMANENT_REDIRECT`.
#
# + code - The response status code
public readonly class StatusPermanentRedirect {
    *Status;
    public STATUS_PERMANENT_REDIRECT code = STATUS_PERMANENT_REDIRECT;
    public isolated function init() {}
}

# Represents the status code of `STATUS_BAD_REQUEST`.
#
# + code - The response status code
public readonly class StatusBadRequest {
    *Status;
    public STATUS_BAD_REQUEST code = STATUS_BAD_REQUEST;
    public isolated function init() {}
}

# Represents the status code of `STATUS_UNAUTHORIZED`.
#
# + code - The response status code
public readonly class StatusUnauthorized {
    *Status;
    public STATUS_UNAUTHORIZED code = STATUS_UNAUTHORIZED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_PAYMENT_REQUIRED`.
#
# + code - The response status code
public readonly class StatusPaymentRequired {
    *Status;
    public STATUS_PAYMENT_REQUIRED code = STATUS_PAYMENT_REQUIRED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_FORBIDDEN`.
#
# + code - The response status code
public readonly class StatusForbidden {
    *Status;
    public STATUS_FORBIDDEN code = STATUS_FORBIDDEN;
    public isolated function init() {}
}

# Represents the status code of `STATUS_NOT_FOUND`.
#
# + code - The response status code
public readonly class StatusNotFound {
    *Status;
    public STATUS_NOT_FOUND code = STATUS_NOT_FOUND;
    public isolated function init() {}
}

# Represents the status code of `STATUS_METHOD_NOT_ALLOWED`.
#
# + code - The response status code
public readonly class StatusMethodNotAllowed {
    *Status;
    public STATUS_METHOD_NOT_ALLOWED code = STATUS_METHOD_NOT_ALLOWED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_NOT_ACCEPTABLE`.
#
# + code - The response status code
public readonly class StatusNotAcceptable {
    *Status;
    public STATUS_NOT_ACCEPTABLE code = STATUS_NOT_ACCEPTABLE;
    public isolated function init() {}
}

# Represents the status code of `STATUS_PROXY_AUTHENTICATION_REQUIRED`.
#
# + code - The response status code
public readonly class StatusProxyAuthenticationRequired {
    *Status;
    public STATUS_PROXY_AUTHENTICATION_REQUIRED code = STATUS_PROXY_AUTHENTICATION_REQUIRED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_REQUEST_TIMEOUT`.
#
# + code - The response status code
public readonly class StatusRequestTimeout {
    *Status;
    public STATUS_REQUEST_TIMEOUT code = STATUS_REQUEST_TIMEOUT;
    public isolated function init() {}
}

# Represents the status code of `STATUS_CONFLICT`.
#
# + code - The response status code
public readonly class StatusConflict {
    *Status;
    public STATUS_CONFLICT code = STATUS_CONFLICT;
    public isolated function init() {}
}

# Represents the status code of `STATUS_GONE`.
#
# + code - The response status code
public readonly class StatusGone {
    *Status;
    public STATUS_GONE code = STATUS_GONE;
    public isolated function init() {}
}

# Represents the status code of `STATUS_LENGTH_REQUIRED`.
#
# + code - The response status code
public readonly class StatusLengthRequired {
    *Status;
    public STATUS_LENGTH_REQUIRED code = STATUS_LENGTH_REQUIRED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_PRECONDITION_FAILED`.
#
# + code - The response status code
public readonly class StatusPreconditionFailed {
    *Status;
    public STATUS_PRECONDITION_FAILED code = STATUS_PRECONDITION_FAILED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_PAYLOAD_TOO_LARGE`.
#
# + code - The response status code
public readonly class StatusPayloadTooLarge {
    *Status;
    public STATUS_PAYLOAD_TOO_LARGE code = STATUS_PAYLOAD_TOO_LARGE;
    public isolated function init() {}
}

# Represents the status code of `STATUS_URI_TOO_LONG`.
#
# + code - The response status code
public readonly class StatusUriTooLong {
    *Status;
    public STATUS_URI_TOO_LONG code = STATUS_URI_TOO_LONG;
    public isolated function init() {}
}

# Represents the status code of `STATUS_UNSUPPORTED_MEDIA_TYPE`.
#
# + code - The response status code
public readonly class StatusUnsupportedMediaType {
    *Status;
    public STATUS_UNSUPPORTED_MEDIA_TYPE code = STATUS_UNSUPPORTED_MEDIA_TYPE;
    public isolated function init() {}
}

# Represents the status code of `STATUS_RANGE_NOT_SATISFIABLE`.
#
# + code - The response status code
public readonly class StatusRangeNotSatisfiable {
    *Status;
    public STATUS_RANGE_NOT_SATISFIABLE code = STATUS_RANGE_NOT_SATISFIABLE;
    public isolated function init() {}
}

# Represents the status code of `STATUS_EXPECTATION_FAILED`.
#
# + code - The response status code
public readonly class StatusExpectationFailed {
    *Status;
    public STATUS_EXPECTATION_FAILED code = STATUS_EXPECTATION_FAILED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_MISDIRECTED_REQUEST`.
#
# + code - The response status code
public readonly class StatusMisdirectedRequest {
    *Status;
    public STATUS_MISDIRECTED_REQUEST code = STATUS_MISDIRECTED_REQUEST;
    public isolated function init() {}
}

# Represents the status code of `STATUS_UNPROCESSABLE_ENTITY`.
#
# + code - The response status code
public readonly class StatusUnprocessableEntity {
    *Status;
    public STATUS_UNPROCESSABLE_ENTITY code = STATUS_UNPROCESSABLE_ENTITY;
    public isolated function init() {}
}

# Represents the status code of `STATUS_LOCKED`.
#
# + code - The response status code
public readonly class StatusLocked {
    *Status;
    public STATUS_LOCKED code = STATUS_LOCKED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_FAILED_DEPENDENCY`.
#
# + code - The response status code
public readonly class StatusFailedDependency {
    *Status;
    public STATUS_FAILED_DEPENDENCY code = STATUS_FAILED_DEPENDENCY;
    public isolated function init() {}
}

# Represents the status code of `STATUS_TOO_EARLY`.
#
# + code - The response status code
public readonly class StatusTooEarly {
    *Status;
    public STATUS_TOO_EARLY code = STATUS_TOO_EARLY;
    public isolated function init() {}
}

# Represents the status code of `STATUS_UPGRADE_REQUIRED`.
#
# + code - The response status code
public readonly class StatusUpgradeRequired {
    *Status;
    public STATUS_UPGRADE_REQUIRED code = STATUS_UPGRADE_REQUIRED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_PRECONDITION_REQUIRED`.
#
# + code - The response status code
public readonly class StatusPreconditionRequired {
    *Status;
    public STATUS_PRECONDITION_REQUIRED code = STATUS_PRECONDITION_REQUIRED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_TOO_MANY_REQUESTS`.
#
# + code - The response status code
public readonly class StatusTooManyRequests {
    *Status;
    public STATUS_TOO_MANY_REQUESTS code = STATUS_TOO_MANY_REQUESTS;
    public isolated function init() {}
}

# Represents the status code of `STATUS_REQUEST_HEADER_FIELDS_TOO_LARGE`.
#
# + code - The response status code
public readonly class StatusRequestHeaderFieldsTooLarge {
    *Status;
    public STATUS_REQUEST_HEADER_FIELDS_TOO_LARGE code = STATUS_REQUEST_HEADER_FIELDS_TOO_LARGE;
    public isolated function init() {}
}

# Represents the status code of `STATUS_UNAVAILABLE_DUE_TO_LEGAL_REASONS`.
#
# + code - The response status code
public readonly class StatusUnavailableDueToLegalReasons {
    *Status;
    public STATUS_UNAVAILABLE_DUE_TO_LEGAL_REASONS code = STATUS_UNAVAILABLE_DUE_TO_LEGAL_REASONS;
    public isolated function init() {}
}

# Represents the status code of `STATUS_INTERNAL_SERVER_ERROR`.
#
# + code - The response status code
public readonly class StatusInternalServerError {
    *Status;
    public STATUS_INTERNAL_SERVER_ERROR code = STATUS_INTERNAL_SERVER_ERROR;
    public isolated function init() {}
}

# Represents the status code of `STATUS_NOT_IMPLEMENTED`.
#
# + code - The response status code
public readonly class StatusNotImplemented {
    *Status;
    public STATUS_NOT_IMPLEMENTED code = STATUS_NOT_IMPLEMENTED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_BAD_GATEWAY`.
#
# + code - The response status code
public readonly class StatusBadGateway {
    *Status;
    public STATUS_BAD_GATEWAY code = STATUS_BAD_GATEWAY;
    public isolated function init() {}
}

# Represents the status code of `STATUS_SERVICE_UNAVAILABLE`.
#
# + code - The response status code
public readonly class StatusServiceUnavailable {
    *Status;
    public STATUS_SERVICE_UNAVAILABLE code = STATUS_SERVICE_UNAVAILABLE;
    public isolated function init() {}
}

# Represents the status code of `STATUS_GATEWAY_TIMEOUT`.
#
# + code - The response status code
public readonly class StatusGatewayTimeout {
    *Status;
    public STATUS_GATEWAY_TIMEOUT code = STATUS_GATEWAY_TIMEOUT;
    public isolated function init() {}
}

# Represents the status code of `STATUS_HTTP_VERSION_NOT_SUPPORTED`.
#
# + code - The response status code
public readonly class StatusHttpVersionNotSupported {
    *Status;
    public STATUS_HTTP_VERSION_NOT_SUPPORTED code = STATUS_HTTP_VERSION_NOT_SUPPORTED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_VARIANT_ALSO_NEGOTIATES`.
#
# + code - The response status code
public readonly class StatusVariantAlsoNegotiates {
    *Status;
    public STATUS_VARIANT_ALSO_NEGOTIATES code = STATUS_VARIANT_ALSO_NEGOTIATES;
    public isolated function init() {}
}

# Represents the status code of `STATUS_INSUFFICIENT_STORAGE`.
#
# + code - The response status code
public readonly class StatusInsufficientStorage {
    *Status;
    public STATUS_INSUFFICIENT_STORAGE code = STATUS_INSUFFICIENT_STORAGE;
    public isolated function init() {}
}

# Represents the status code of `STATUS_LOOP_DETECTED`.
#
# + code - The response status code
public readonly class StatusLoopDetected {
    *Status;
    public STATUS_LOOP_DETECTED code = STATUS_LOOP_DETECTED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_NOT_EXTENDED`.
#
# + code - The response status code
public readonly class StatusNotExtended {
    *Status;
    public STATUS_NOT_EXTENDED code = STATUS_NOT_EXTENDED;
    public isolated function init() {}
}

# Represents the status code of `STATUS_NETWORK_AUTHENTICATION_REQUIRED`.
#
# + code - The response status code
public readonly class StatusNetworkAuthenticationRequired {
    *Status;
    public STATUS_NETWORK_AUTHENTICATION_REQUIRED code = STATUS_NETWORK_AUTHENTICATION_REQUIRED;
    public isolated function init() {}
}

// Status code record types

# The default status code response record.
#
# + status - The response status code object
public type DefaultStatusCodeResponse record {|
    *CommonResponse;
    readonly DefaultStatus status;
|};

# The status code response record of `Continue`.
#
# + status - The response status code obj
public type Continue record {|
    *CommonResponse;
    readonly StatusContinue status = new;
|};

# The status code response record of `SwitchingProtocols`.
#
# + status - The response status code obj
public type SwitchingProtocols record {|
    *CommonResponse;
    readonly StatusSwitchingProtocols status = new;
|};

# The status code response record of `Processing`.
#
# + status - The response status code obj
public type Processing record {|
    *CommonResponse;
    readonly StatusProcessing status = new;
|};

# The status code response record of `EarlyHints`.
#
# + status - The response status code obj
public type EarlyHints record {|
    *CommonResponse;
    readonly StatusEarlyHints status = new;
|};

# The status code response record of `Ok`.
#
# + status - The response status code obj
public type Ok record {|
    *CommonResponse;
    readonly StatusOK status = new;
|};

# The status code response record of `Created`.
#
# + status - The response status code obj
public type Created record {|
    *CommonResponse;
    readonly StatusCreated status = new;
|};

# The status code response record of `Accepted`.
#
# + status - The response status code obj
public type Accepted record {|
    *CommonResponse;
    readonly StatusAccepted status = new;
|};

# The status code response record of `NonAuthoritativeInformation`.
#
# + status - The response status code obj
public type NonAuthoritativeInformation record {|
    *CommonResponse;
    readonly StatusNonAuthoritativeInformation status = new;
|};

# The status code response record of `NoContent`.
#
# + headers - The response headers
# + status - The response status code obj
public type NoContent record {|
    map<string|int|boolean|string[]|int[]|boolean[]> headers?;
    readonly StatusNoContent status = new;
|};

# The status code response record of `ResetContent`.
#
# + status - The response status code obj
public type ResetContent record {|
    *CommonResponse;
    readonly StatusResetContent status = new;
|};

# The status code response record of `PartialContent`.
#
# + status - The response status code obj
public type PartialContent record {|
    *CommonResponse;
    readonly StatusPartialContent status = new;
|};

# The status code response record of `MultiStatus`.
#
# + status - The response status code obj
public type MultiStatus record {|
    *CommonResponse;
    readonly StatusMultiStatus status = new;
|};

# The status code response record of `AlreadyReported`.
#
# + status - The response status code obj
public type AlreadyReported record {|
    *CommonResponse;
    readonly StatusAlreadyReported status = new;
|};

# The status code response record of `IMUsed`.
#
# + status - The response status code obj
public type IMUsed record {|
    *CommonResponse;
    readonly StatusIMUsed status = new;
|};

# The status code response record of `MultipleChoices`.
#
# + status - The response status code obj
public type MultipleChoices record {|
    *CommonResponse;
    readonly StatusMultipleChoices status = new;
|};

# The status code response record of `MovedPermanently`.
#
# + status - The response status code obj
public type MovedPermanently record {|
    *CommonResponse;
    readonly StatusMovedPermanently status = new;
|};

# The status code response record of `Found`.
#
# + status - The response status code obj
public type Found record {|
    *CommonResponse;
    readonly StatusFound status = new;
|};

# The status code response record of `SeeOther`.
#
# + status - The response status code obj
public type SeeOther record {|
    *CommonResponse;
    readonly StatusSeeOther status = new;
|};

# The status code response record of `NotModified`.
#
# + status - The response status code obj
public type NotModified record {|
    *CommonResponse;
    readonly StatusNotModified status = new;
|};

# The status code response record of `UseProxy`.
#
# + status - The response status code obj
public type UseProxy record {|
    *CommonResponse;
    readonly StatusUseProxy status = new;
|};

# The status code response record of `TemporaryRedirect`.
#
# + status - The response status code obj
public type TemporaryRedirect record {|
    *CommonResponse;
    readonly StatusTemporaryRedirect status = new;
|};

# The status code response record of `PermanentRedirect`.
#
# + status - The response status code obj
public type PermanentRedirect record {|
    *CommonResponse;
    readonly StatusPermanentRedirect status = new;
|};

# The status code response record of `BadRequest`.
#
# + status - The response status code obj
public type BadRequest record {|
    *CommonResponse;
    readonly StatusBadRequest status = new;
|};

# The status code response record of `Unauthorized`.
#
# + status - The response status code obj
public type Unauthorized record {|
    *CommonResponse;
    readonly StatusUnauthorized status = new;
|};

# The status code response record of `PaymentRequired`.
#
# + status - The response status code obj
public type PaymentRequired record {|
    *CommonResponse;
    readonly StatusPaymentRequired status = new;
|};

# The status code response record of `Forbidden`.
#
# + status - The response status code obj
public type Forbidden record {|
    *CommonResponse;
    readonly StatusForbidden status = new;
|};

# The status code response record of `NotFound`.
#
# + status - The response status code obj
public type NotFound record {|
    *CommonResponse;
    readonly StatusNotFound status = new;
|};

# The status code response record of `MethodNotAllowed`.
#
# + status - The response status code obj
public type MethodNotAllowed record {|
    *CommonResponse;
    readonly StatusMethodNotAllowed status = new;
|};

# The status code response record of `NotAcceptable`.
#
# + status - The response status code obj
public type NotAcceptable record {|
    *CommonResponse;
    readonly StatusNotAcceptable status = new;
|};

# The status code response record of `ProxyAuthenticationRequired`.
#
# + status - The response status code obj
public type ProxyAuthenticationRequired record {|
    *CommonResponse;
    readonly StatusProxyAuthenticationRequired status = new;
|};

# The status code response record of `RequestTimeout`.
#
# + status - The response status code obj
public type RequestTimeout record {|
    *CommonResponse;
    readonly StatusRequestTimeout status = new;
|};

# The status code response record of `Conflict`.
#
# + status - The response status code obj
public type Conflict record {|
    *CommonResponse;
    readonly StatusConflict status = new;
|};

# The status code response record of `Gone`.
#
# + status - The response status code obj
public type Gone record {|
    *CommonResponse;
    readonly StatusGone status = new;
|};

# The status code response record of `LengthRequired`.
#
# + status - The response status code obj
public type LengthRequired record {|
    *CommonResponse;
    readonly StatusLengthRequired status = new;
|};

# The status code response record of `PreconditionFailed`.
#
# + status - The response status code obj
public type PreconditionFailed record {|
    *CommonResponse;
    readonly StatusPreconditionFailed status = new;
|};

# The status code response record of `PayloadTooLarge`.
#
# + status - The response status code obj
public type PayloadTooLarge record {|
    *CommonResponse;
    readonly StatusPayloadTooLarge status = new;
|};

# The status code response record of `UriTooLong`.
#
# + status - The response status code obj
public type UriTooLong record {|
    *CommonResponse;
    readonly StatusUriTooLong status = new;
|};

# The status code response record of `UnsupportedMediaType`.
#
# + status - The response status code obj
public type UnsupportedMediaType record {|
    *CommonResponse;
    readonly StatusUnsupportedMediaType status = new;
|};

# The status code response record of `RangeNotSatisfiable`.
#
# + status - The response status code obj
public type RangeNotSatisfiable record {|
    *CommonResponse;
    readonly StatusRangeNotSatisfiable status = new;
|};

# The status code response record of `ExpectationFailed`.
#
# + status - The response status code obj
public type ExpectationFailed record {|
    *CommonResponse;
    readonly StatusExpectationFailed status = new;
|};

# The status code response record of `MisdirectedRequest`.
#
# + status - The response status code obj
public type MisdirectedRequest record {|
    *CommonResponse;
    readonly StatusMisdirectedRequest status = new;
|};

# The status code response record of `UnprocessableEntity`.
#
# + status - The response status code obj
public type UnprocessableEntity record {|
    *CommonResponse;
    readonly StatusUnprocessableEntity status = new;
|};

# The status code response record of `Locked`.
#
# + status - The response status code obj
public type Locked record {|
    *CommonResponse;
    readonly StatusLocked status = new;
|};

# The status code response record of `FailedDependency`.
#
# + status - The response status code obj
public type FailedDependency record {|
    *CommonResponse;
    readonly StatusFailedDependency status = new;
|};

# The status code response record of `TooEarly`.
#
# + status - The response status code obj
public type TooEarly record {|
    *CommonResponse;
    readonly StatusTooEarly status = new;
|};

# The status code response record of `PreconditionRequired`.
#
# + status - The response status code obj
public type PreconditionRequired record {|
    *CommonResponse;
    readonly StatusPreconditionRequired status = new;
|};

# The status code response record of `UnavailableDueToLegalReasons`.
#
# + status - The response status code obj
public type UnavailableDueToLegalReasons record {|
    *CommonResponse;
    readonly StatusUnavailableDueToLegalReasons status = new;
|};

# The status code response record of `UpgradeRequired`.
#
# + status - The response status code obj
public type UpgradeRequired record {|
    *CommonResponse;
    readonly StatusUpgradeRequired status = new;
|};

# The status code response record of `TooManyRequests`.
#
# + status - The response status code obj
public type TooManyRequests record {|
    *CommonResponse;
    readonly StatusTooManyRequests status = new;
|};

# The status code response record of `RequestHeaderFieldsTooLarge`.
#
# + status - The response status code obj
public type RequestHeaderFieldsTooLarge record {|
    *CommonResponse;
    readonly StatusRequestHeaderFieldsTooLarge status = new;
|};

# The status code response record of `InternalServerError`.
#
# + status - The response status code obj
public type InternalServerError record {|
    *CommonResponse;
    readonly StatusInternalServerError status = new;
|};

# The status code response record of `NotImplemented`.
#
# + status - The response status code obj
public type NotImplemented record {|
    *CommonResponse;
    readonly StatusNotImplemented status = new;
|};

# The status code response record of `BadGateway`.
#
# + status - The response status code obj
public type BadGateway record {|
    *CommonResponse;
    readonly StatusBadGateway status = new;
|};

# The status code response record of `ServiceUnavailable`.
#
# + status - The response status code obj
public type ServiceUnavailable record {|
    *CommonResponse;
    readonly StatusServiceUnavailable status = new;
|};

# The status code response record of `GatewayTimeout`.
#
# + status - The response status code obj
public type GatewayTimeout record {|
    *CommonResponse;
    readonly StatusGatewayTimeout status = new;
|};

# The status code response record of `HttpVersionNotSupported`.
#
# + status - The response status code obj
public type HttpVersionNotSupported record {|
    *CommonResponse;
    readonly StatusHttpVersionNotSupported status = new;
|};

# The status code response record of `VariantAlsoNegotiates`.
#
# + status - The response status code obj
public type VariantAlsoNegotiates record {|
    *CommonResponse;
    readonly StatusVariantAlsoNegotiates status = new;
|};

# The status code response record of `InsufficientStorage`.
#
# + status - The response status code obj
public type InsufficientStorage record {|
    *CommonResponse;
    readonly StatusInsufficientStorage status = new;
|};

# The status code response record of `LoopDetected`.
#
# + status - The response status code obj
public type LoopDetected record {|
    *CommonResponse;
    readonly StatusLoopDetected status = new;
|};

# The status code response record of `NotExtended`.
#
# + status - The response status code obj
public type NotExtended record {|
    *CommonResponse;
    readonly StatusNotExtended status = new;
|};

# The status code response record of `NetworkAuthorizationRequired`.
#
# + status - The response status code obj
# # Deprecated
# This record is deprecated. Please use `NetworkAuthenticationRequired` instead.
public type NetworkAuthorizationRequired record {|
    *CommonResponse;
    readonly StatusNetworkAuthenticationRequired status = new;
|};

# The status code response record of `NetworkAuthenticationRequired`.
#
# + status - The response status code obj
public type NetworkAuthenticationRequired record {|
    *CommonResponse;
    readonly StatusNetworkAuthenticationRequired status = new;
|};
