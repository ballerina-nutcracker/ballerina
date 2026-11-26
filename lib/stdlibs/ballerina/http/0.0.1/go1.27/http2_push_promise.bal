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

# Represents an HTTP/2 `PUSH_PROMISE` frame.
#
# + path - The resource path
# + method - The HTTP method
public class PushPromise {

    public string path;
    public string method;

    # Constructs an `http:PushPromise` from a given path and a method.
    #
    # + path - The resource path
    # + method - The HTTP method
    public isolated function init(string path = "/", string method = "GET") {
        self.path = path;
        self.method = method;
        externPromiseInit(self);
    }

    # Checks whether the requested header exists.
    #
    # + headerName - The header name
    # + return - A `boolean` representing the existence of a given header
    public isolated function hasHeader(string headerName) returns boolean {
        return externPromiseHasHeader(self, headerName);
    }

    # Returns the header value with the specified header name.
    # If there are more than one header value for the specified header name, the first value is returned.
    #
    # + headerName - The header name
    # + return - The header value or `()` if there is no such header
    public isolated function getHeader(string headerName) returns string {
        return externPromiseGetHeader(self, headerName);
    }

    # Gets transport headers from the `PushPromise`.
    #
    # + headerName - The header name
    # + return - The array of header values
    public isolated function getHeaders(string headerName) returns string[] {
        return externPromiseGetHeaders(self, headerName);
    }

    # Adds the specified key/value pair as an HTTP header to the `http:PushPromise`. In the case of the `Content-Type`
    # header, the existing value is replaced with the specified value.
    #
    # + headerName - The header name
    # + headerValue - The header value
    public isolated function addHeader(string headerName, string headerValue) {
        if headerName.equalsIgnoreCaseAscii("content-type") {
            return externPromiseSetHeader(self, headerName, headerValue);
        }
        return externPromiseAddHeader(self, headerName, headerValue);
    }

    # Sets the value of a transport header in the `http:PushPromise`.
    #
    # + headerName - The header name
    # + headerValue - The header value
    public isolated function setHeader(string headerName, string headerValue) {
        return externPromiseSetHeader(self, headerName, headerValue);
    }

    # Removes a transport header from the `http:PushPromise`.
    #
    # + headerName - The header name
    public isolated function removeHeader(string headerName) {
        return externPromiseRemoveHeader(self, headerName);
    }

    # Removes all transport headers from the `http:PushPromise`.
    public isolated function removeAllHeaders() {
        return externPromiseRemoveAllHeaders(self);
    }

    # Gets all transport header names from the `http:PushPromise`.
    #
    # + return - An array of all transport header names
    public isolated function getHeaderNames() returns string[] {
        return externPromiseGetHeaderNames(self);
    }
}

isolated function externPromiseInit(PushPromise promise) = external;

isolated function externPromiseHasHeader(PushPromise promise, string headerName) returns boolean = external;

isolated function externPromiseGetHeader(PushPromise promise, string headerName) returns string = external;

isolated function externPromiseGetHeaders(PushPromise promise, string headerName) returns string[] = external;

isolated function externPromiseAddHeader(PushPromise promise, string headerName, string headerValue) = external;

isolated function externPromiseSetHeader(PushPromise promise, string headerName, string headerValue) = external;

isolated function externPromiseRemoveHeader(PushPromise promise, string headerName) = external;

isolated function externPromiseRemoveAllHeaders(PushPromise promise) = external;

isolated function externPromiseGetHeaderNames(PushPromise promise) returns string[] = external;
