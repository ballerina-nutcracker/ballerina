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
import ballerina/http.httpscerr;
import ballerina/io;

public function main() {
    error generic = error http:GenericClientError("generic");
    io:println(generic is http:ClientError, " ", generic is http:Error, " ", generic is http:ListenerError); // @output true true false

    error noContent = error http:NoContentError("No content");
    io:println(noContent is http:ClientError, " ", noContent is http:GenericClientError); // @output true false

    error header = error http:HeaderNotFoundError("Http header does not exist");
    io:println(header is http:Error, " ", header is http:ClientError); // @output true false

    error listenerError = error http:GenericListenerError("listener");
    io:println(listenerError is http:ListenerError, " ", listenerError is http:ClientError); // @output true false

    error notFound = error httpscerr:NotFoundError("not found");
    io:println(notFound is httpscerr:'4XXStatusCodeError, " ", notFound is httpscerr:StatusCodeError, " ", notFound is httpscerr:'5XXStatusCodeError); // @output true true false

    error defaultStatus = error httpscerr:DefaultStatusCodeError("teapot", statusCode = 418);
    io:println(defaultStatus is httpscerr:StatusCodeError, " ", defaultStatus is httpscerr:NotFoundError); // @output true false

    error queryValidation = error http:QueryParameterValidationError("bad query");
    io:println(queryValidation is http:QueryParameterBindingError, " ", queryValidation is httpscerr:BadRequestError, " ", queryValidation is http:ListenerError); // @output true true true

    error pathBinding = error http:PathParameterBindingError("bad path");
    io:println(pathBinding is http:QueryParameterBindingError, " ", pathBinding is httpscerr:BadRequestError); // @output false true

    error resourceNotFound = error http:ResourceNotFoundError("no resource");
    io:println(resourceNotFound is httpscerr:NotFoundError, " ", resourceNotFound is http:ResourceDispatchingError, " ", resourceNotFound is http:RequestDispatchingError); // @output true true true

    error authn = error http:ListenerAuthnError("unauthorized");
    io:println(authn is httpscerr:UnauthorizedError, " ", authn is http:ListenerAuthError, " ", authn is http:ListenerAuthzError); // @output true true false

    error interceptor = error http:InterceptorReturnError("interceptor");
    io:println(interceptor is httpscerr:InternalServerErrorError, " ", interceptor is http:ListenerError); // @output true true

    error clientRequest = error http:ClientRequestError("Not Found", statusCode = 404, headers = {}, body = ());
    io:println(clientRequest is http:ApplicationResponseError, " ", clientRequest is http:ClientError, " ", clientRequest is http:RemoteServerError); // @output true true false

    error remoteServer = error http:RemoteServerError("Internal Server Error", statusCode = 500, headers = {}, body = ());
    io:println(remoteServer is http:ApplicationResponseError, " ", remoteServer is http:ClientRequestError); // @output true false

    error statusBinding = error http:StatusCodeBindingClientRequestError("Not Found", statusCode = 404, headers = {}, body = (), fromDefaultStatusCodeMapping = true);
    io:println(statusBinding is http:StatusCodeResponseBindingError, " ", statusBinding is http:ClientRequestError, " ", statusBinding is http:RemoteServerError); // @output true true false

    error loadBalance = error http:LoadBalanceActionError("all failed");
    io:println(loadBalance is http:ResiliencyError, " ", loadBalance is http:ClientError); // @output true true
}
