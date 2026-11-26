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

import ballerina/http;

class CustomListener {
    public function attach(service object {} svc, string[] attachPoint = []) returns error? {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(service object {} svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

service /invalid on new http:Listener(19222) {
    resource function post tooMany(http:Request first, http:Request second) { // @error
        var _ = first;
        var _ = second;
    }
}

service /caller on new http:Listener(19227) {
    resource function get twoCallers(http:Caller first, http:Caller second) { // @error
        var _ = first;
        var _ = second;
    }

    resource function get callerString(http:Caller caller) returns string { // @error
        var _ = caller;
        return "x";
    }

    resource function get callerUnion(http:Caller caller) returns string|error? { // @error
        var _ = caller;
        return "x";
    }

    resource function post callerTwoRequests(http:Caller caller, http:Request first, http:Request second) returns string { // @error
        var _ = caller;
        var _ = first;
        var _ = second;
        return "x";
    }

    resource function get callerError(http:Caller caller) returns error? {
        var _ = caller;
    }
}

service /custom on new CustomListener() {
    resource function get ignored(int value) {
        var _ = value;
    }
}

service /mixed on new CustomListener(), new http:Listener(19223) {
    resource function patch checked(http:Request first, http:Request second) { // @error
        var _ = first;
        var _ = second;
    }
}

@http:ServiceConfig {
    mediaTypeSubtypePrefix: "vnd.a+json", // @error
    basePath: "/contract" // @error
}
service /prefixSuffix on new http:Listener(19224) {
    resource function get .() returns string => "x";
}

@http:ServiceConfig {
    mediaTypeSubtypePrefix: "vnd/a" // @error
}
service /prefixInvalid on new http:Listener(19225) {
    resource function get .() returns string => "x";
}

const PREFIX = "vnd/a";

@http:ServiceConfig {
    mediaTypeSubtypePrefix: PREFIX
}
service /prefixConstant on new http:Listener(19226) {
    resource function get .() returns string => "x";
}

service class TwoKinds {
    *http:RequestInterceptor;
    *http:ResponseInterceptor; // @error

    resource function get a() returns string => "a";
}

service class NoResource { // @error
    *http:RequestInterceptor;
}

service class Remote {
    *http:RequestInterceptor;

    resource function get a() returns string => "a";

    remote function r() { // @error
    }
}

service class TwoResources {
    *http:RequestInterceptor;

    resource function get a() returns string => "a";

    resource function get b() returns string => "b"; // @error
}

service class Annotated {
    *http:RequestInterceptor;

    @http:ResourceConfig {} // @error
    resource function get a() returns string => "a";
}

service class BadReturn {
    *http:RequestInterceptor;

    resource function get a() returns typedesc<int> => int; // @error
}

service class ReturnAnnotated {
    *http:RequestInterceptor;

    resource function get a() returns @http:Payload string => "a"; // @error
}

service class TwoContexts {
    *http:RequestInterceptor;

    resource function get a(http:RequestContext first, http:RequestContext second) returns http:NextService|error? { // @error
        var _ = second;
        return first.next();
    }
}

service /contexts on new http:Listener(19228) {
    resource function get a(http:RequestContext first, http:RequestContext second) { // @error
        var _ = first;
        var _ = second;
    }
}

service class ResponseWithResources {
    *http:ResponseInterceptor;

    resource function get a() returns string => "a"; // @error
    resource function get b() returns string => "b"; // @error

    remote function interceptResponse() returns http:NextService|error? {
        return;
    }
}

service class ResponseMisnamed { // @error
    *http:ResponseInterceptor;

    remote function intercept() { // @error
    }
}

service class ResponseParams {
    *http:ResponseInterceptor;

    remote function interceptResponse(http:Response first, http:Response second, string s, error e, // @error
            http:Caller c, http:Caller c2, http:Request r, http:Request r2, http:RequestContext x, // @error
            http:RequestContext x2) returns int|typedesc<int> { // @error
        var _ = [first, second, s, e, c, c2, r, r2, x, x2];
        return 1;
    }
}

service class ResponseErrorWithoutError {
    *http:ResponseErrorInterceptor;

    remote function interceptResponseError(http:Response res) returns http:Response { // @error
        return res;
    }
}

service class ResponseErrorTwoErrors {
    *http:ResponseErrorInterceptor;

    remote function interceptResponseError(error first, error second) returns @http:Payload {} http:Response { // @error
        var _ = [first, second];
        return new;
    }
}

service class ResponseErrorWrongName {
    *http:ResponseErrorInterceptor;

    remote function interceptResponse(error e) returns http:Response { // @error
        string _ = e.message();
        return new;
    }

    remote function interceptResponseError(error e) returns http:Response {
        string _ = e.message();
        return new;
    }
}

service class NoRemote { // @error
    *http:ResponseErrorInterceptor;
}

service class RequestErrorGet {
    *http:RequestErrorInterceptor;

    resource function get [string... path](error e) returns http:NextService|error? { // @error
        var _ = [path, e];
        return;
    }
}

service class RequestErrorPath {
    *http:RequestErrorInterceptor;

    resource function 'default foo(error e) returns http:NextService|error? { // @error
        string _ = e.message();
        return;
    }
}

service class RequestErrorIntRest {
    *http:RequestErrorInterceptor;

    resource function 'default [int... path](error e) returns http:NextService|error? { // @error
        var _ = [path, e];
        return;
    }
}

service class RequestErrorWithoutError {
    *http:RequestErrorInterceptor;

    resource function 'default [string... path]() returns http:NextService|error? { // @error
        var _ = path;
        return;
    }
}

service class RequestErrorTwoErrors {
    *http:RequestErrorInterceptor;

    resource function 'default [string... path](error e, error f) returns http:NextService|error? { // @error
        var _ = [path, e, f];
        return;
    }

    remote function r() { // @error
    }
}

type NamedError distinct error;

service class ResponseErrorNamedError {
    *http:ResponseErrorInterceptor;

    remote function interceptResponseError(http:ListenerError e) returns http:Response { // @error
        string _ = e.message();
        return new;
    }
}

service class RequestErrorNamedError {
    *http:RequestErrorInterceptor;

    resource function 'default [string... path](NamedError e) returns http:NextService|error? { // @error
        var _ = path;
        string _ = e.message();
        return;
    }
}

service /returnAnnotations on new http:Listener(19233) {
    resource function get cacheError() returns @http:Cache error { // @error
        return error("cache");
    }

    resource function get payloadError() returns @http:Payload {mediaType: "text/plain"} error { // @error
        return error("payload");
    }

    resource function get both(http:Caller caller) returns @http:Payload {mediaType: "text/plain"} @http:Cache error? { // @error
        var _ = caller;
    }

    resource function get namedError() returns @http:Cache http:ListenerError { // @error
        return error http:GenericListenerError("named");
    }

    resource function get cacheUnion() returns @http:Cache string|error => "union";

    resource function get cacheNotFound() returns @http:Cache http:NotFound => {};
}
