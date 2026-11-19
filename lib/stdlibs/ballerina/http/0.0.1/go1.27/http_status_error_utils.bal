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


import ballerina/log;
import http.httpscerr;

# Represents the structure of the HTTP error payload.
#
# + timestamp - Timestamp of the error
# + status - Relevant HTTP status code
# + reason - Reason phrase
# + message - Error message
# + path - Request path
# + method - Method type of the request
public type ErrorPayload record {
    string timestamp;
    int status;
    string reason;
    string message;
    string path;
    string method;
};

// Returns the response a resource's returned error is sent as; the native listener dispatcher calls it.
isolated function toErrorResponse(error err, string path, string method) returns Response {
    if err !is InternalError {
        log:printError("unhandled error returned from the service", err, path = path, method = method);
    }
    Response response = new;
    map<anydata> detail = errorDetail(err);
    if err is ApplicationResponseError {
        response.statusCode = <int>detail.get("statusCode");
    } else {
        response.statusCode = getStatusCode(err);
        if err !is httpscerr:StatusCodeError {
            response.setErrorPayload(err.message(), path, method);
            return response;
        }
    }
    // Set first, as for a status code record, so a Content-Type in the detail wins over the payload's.
    anydata headers = detail["headers"];
    if headers is map<string|string[]> {
        setStatusCodeHeaders(headers, response);
    }
    anydata body = detail["body"];
    if body is () {
        response.setErrorPayload(err.message(), path, method);
    } else {
        setStatusCodePayload(body, response, ());
    }
    return response;
}

// Each test casts err so it is not narrowed, which keeps compile time linear (#1745).
isolated function getStatusCode(error err) returns int {
    if (<error>err) is httpscerr:BadRequestError {
        return 400;
    } else if (<error>err) is httpscerr:UnauthorizedError {
        return 401;
    } else if (<error>err) is httpscerr:PaymentRequiredError {
        return 402;
    } else if (<error>err) is httpscerr:ForbiddenError {
        return 403;
    } else if (<error>err) is httpscerr:NotFoundError {
        return 404;
    } else if (<error>err) is httpscerr:MethodNotAllowedError {
        return 405;
    } else if (<error>err) is httpscerr:NotAcceptableError {
        return 406;
    } else if (<error>err) is httpscerr:ProxyAuthenticationRequiredError {
        return 407;
    } else if (<error>err) is httpscerr:RequestTimeoutError {
        return 408;
    } else if (<error>err) is httpscerr:ConflictError {
        return 409;
    } else if (<error>err) is httpscerr:GoneError {
        return 410;
    } else if (<error>err) is httpscerr:LengthRequiredError {
        return 411;
    } else if (<error>err) is httpscerr:PreconditionFailedError {
        return 412;
    } else if (<error>err) is httpscerr:PayloadTooLargeError {
        return 413;
    } else if (<error>err) is httpscerr:URITooLongError {
        return 414;
    } else if (<error>err) is httpscerr:UnsupportedMediaTypeError {
        return 415;
    } else if (<error>err) is httpscerr:RangeNotSatisfiableError {
        return 416;
    } else if (<error>err) is httpscerr:ExpectationFailedError {
        return 417;
    } else if (<error>err) is httpscerr:MisdirectedRequestError {
        return 421;
    } else if (<error>err) is httpscerr:UnprocessableEntityError {
        return 422;
    } else if (<error>err) is httpscerr:LockedError {
        return 423;
    } else if (<error>err) is httpscerr:FailedDependencyError {
        return 424;
    } else if (<error>err) is httpscerr:UpgradeRequiredError {
        return 426;
    } else if (<error>err) is httpscerr:PreconditionRequiredError {
        return 428;
    } else if (<error>err) is httpscerr:TooManyRequestsError {
        return 429;
    } else if (<error>err) is httpscerr:RequestHeaderFieldsTooLargeError {
        return 431;
    } else if (<error>err) is httpscerr:UnavailableDueToLegalReasonsError {
        return 451;
    } else if (<error>err) is httpscerr:InternalServerErrorError {
        return 500;
    } else if (<error>err) is httpscerr:NotImplementedError {
        return 501;
    } else if (<error>err) is httpscerr:BadGatewayError {
        return 502;
    } else if (<error>err) is httpscerr:ServiceUnavailableError {
        return 503;
    } else if (<error>err) is httpscerr:GatewayTimeoutError {
        return 504;
    } else if (<error>err) is httpscerr:HTTPVersionNotSupportedError {
        return 505;
    } else if (<error>err) is httpscerr:VariantAlsoNegotiatesError {
        return 506;
    } else if (<error>err) is httpscerr:InsufficientStorageError {
        return 507;
    } else if (<error>err) is httpscerr:LoopDetectedError {
        return 508;
    } else if (<error>err) is httpscerr:NotExtendedError {
        return 510;
    } else if (<error>err) is httpscerr:NetworkAuthenticationRequiredError {
        return 511;
    } else if (<error>err) is httpscerr:DefaultStatusCodeError {
        anydata statusCode = errorDetail(err)["statusCode"];
        if statusCode is int {
            return statusCode;
        }
        error? cause = err.cause();
        if cause is error {
            return getStatusCode(cause);
        }
    }
    return 500;
}

isolated function errorDetail(error err) returns map<anydata> = external;
