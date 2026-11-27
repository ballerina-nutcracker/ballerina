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

import ballerina/mime;
import http.httpscerr;

# The types of messages that are accepted by HTTP `listener` when sending out the outbound response.
# The `stream` members of jBallerina's `ResponseMessage` are not supported in this implementation.
public type ResponseMessage anydata|Response|mime:Entity[];

# Presents a read-only view of the remote address.
#
# + host - The remote host name
# + port - The remote port
# + ip - The remote IP address
public type Remote record {|
    string host;
    int port;
    string ip;
|};

# Presents a read-only view of the local address.
#
# + host - The local host name
# + port - The local port
# + ip - The local IP address
public type Local record {|
    string host;
    int port;
    string ip;
|};

# The caller actions for responding to client requests.
#
# + remoteAddress - The remote address
# + localAddress - The local address
# + protocol - The protocol associated with the service endpoint
public isolated client class Caller {
    public final readonly & Remote remoteAddress;
    public final readonly & Local localAddress;
    public final string protocol;
    private final string? resourceAccessor;

    // The addresses arrive read-only because cloneReadOnly() is not available yet (#1337).
    isolated function init(readonly & Remote remoteAddress, readonly & Local localAddress, string protocol,
            string? resourceAccessor) {
        self.remoteAddress = remoteAddress;
        self.localAddress = localAddress;
        self.protocol = protocol;
        self.resourceAccessor = resourceAccessor;
    }

    # Sends the outbound response to the caller.
    #
    # + message - The outbound response or status code response or error or any allowed payload
    # + return - An `http:ListenerError` if failed to respond or else `()`
    remote isolated function respond(ResponseMessage|StatusCodeResponse|error message = ()) returns ListenerError? {
        if message is StatusCodeResponse {
            return nativeRespond(self, createStatusCodeResponse(message));
        }
        if message is ClientConnectorError {
            return nativeRespondError(self, error httpscerr:BadGatewayError(getClientConnectorErrorCause(message)));
        }
        if message is error {
            return nativeRespondError(self, message);
        }
        Response response = check buildResponse(message, self.resourceAccessor);
        return nativeRespond(self, response);
    }

    # Sends a `100-continue` response to the caller.
    #
    # + return - An `http:ListenerError` if failed to send the `100-continue` response or else `()`
    remote isolated function 'continue() returns ListenerError? {
        Response res = new;
        res.statusCode = STATUS_CONTINUE;
        return self->respond(res);
    }

    # Sends a redirect response to the user with the specified redirection status code.
    #
    # + response - Response to be sent to the caller
    # + code - The redirect status code to be sent
    # + locations - An array of URLs to which the caller can redirect to
    # + return - An `http:ListenerError` if failed to send the redirect response or else `()`
    remote isolated function redirect(Response response, RedirectCode code, string[] locations) returns ListenerError? {
        response.statusCode = code;
        string locationsStr = "";
        foreach int i in 0 ..< locations.length() {
            locationsStr += i == 0 ? locations[i] : "," + locations[i];
        }
        response.setHeader("Location", locationsStr);
        return self->respond(response);
    }

    # Gets the hostname from the remote address. This method may trigger a DNS reverse lookup if the address was created
    # with a literal IP address.
    # ```ballerina
    # string? remoteHost = caller.getRemoteHostName();
    # ```
    #
    # + return - The hostname of the address or else `()` if it is unresolved
    public isolated function getRemoteHostName() returns string? {
        return nativeGetRemoteHostName(self);
    }
}

// The dispatcher calls this to build an http:Caller parameter, and reads its signature to
// learn the Caller type, which a native cannot name directly.
isolated function createCaller(string remoteHost, int remotePort, string remoteIp, string localHost, int localPort,
        string localIp, string? resourceAccessor) returns Caller {
    readonly & Remote remoteAddress = {host: remoteHost, port: remotePort, ip: remoteIp};
    readonly & Local localAddress = {host: localHost, port: localPort, ip: localIp};
    return new (remoteAddress, localAddress, "http", resourceAccessor);
}

isolated function getClientConnectorErrorCause(error err) returns string {
    error? cause = err.cause();
    if cause is error {
        return string `${err.message()}: ${getClientConnectorErrorCause(cause)}`;
    }
    return err.message();
}

// toResponse converts what a stage of a service with interceptors returned into the response the
// response interceptors receive; a () return becomes an empty response with nilStatus.
isolated function toResponse(any message, string? resourceAccessor, int nilStatus) returns Response|error {
    if message is () {
        Response response = new;
        response.statusCode = nilStatus;
        return response;
    }
    if message is StatusCodeResponse {
        return createStatusCodeResponse(message);
    }
    if message is ResponseMessage {
        return buildResponse(message, resourceAccessor);
    }
    return error InitializingOutboundResponseError("unexpected return type from resource method");
}

isolated function buildResponse(ResponseMessage message, string? resourceAccessor) returns Response|ListenerError {
    if message is () {
        return new Response();
    }
    if message is Response {
        return message;
    }
    Response response = new;
    if message is string {
        response.setTextPayload(message);
    } else if message is xml {
        response.setXmlPayload(message);
    } else if message is byte[] {
        response.setBinaryPayload(message);
    } else if message is mime:Entity[] {
        response.setBodyParts(message);
    } else {
        error? result = response.setAnydataPayload(message);
        if result is error {
            return error InitializingOutboundResponseError("json conversion error: " + result.message(), result);
        }
    }
    if resourceAccessor == "POST" {
        response.statusCode = STATUS_CREATED;
    }
    return response;
}

isolated function nativeRespond(Caller caller, Response response) returns ListenerError? = external;

isolated function nativeRespondError(Caller caller, error err) returns ListenerError? = external;

isolated function nativeGetRemoteHostName(Caller caller) returns string? = external;

# Defines the HTTP redirect codes as a type.
public type RedirectCode REDIRECT_MULTIPLE_CHOICES_300|REDIRECT_MOVED_PERMANENTLY_301|REDIRECT_FOUND_302|REDIRECT_SEE_OTHER_303|
REDIRECT_NOT_MODIFIED_304|REDIRECT_USE_PROXY_305|REDIRECT_TEMPORARY_REDIRECT_307|REDIRECT_PERMANENT_REDIRECT_308;

# Represents the HTTP redirect status code `300 - Multiple Choices`.
public const REDIRECT_MULTIPLE_CHOICES_300 = 300;
# Represents the HTTP redirect status code `301 - Moved Permanently`.
public const REDIRECT_MOVED_PERMANENTLY_301 = 301;
# Represents the HTTP redirect status code `302 - Found`.
public const REDIRECT_FOUND_302 = 302;
# Represents the HTTP redirect status code `303 - See Other`.
public const REDIRECT_SEE_OTHER_303 = 303;
# Represents the HTTP redirect status code `304 - Not Modified`.
public const REDIRECT_NOT_MODIFIED_304 = 304;
# Represents the HTTP redirect status code `305 - Use Proxy`.
public const REDIRECT_USE_PROXY_305 = 305;
# Represents the HTTP redirect status code `307 - Temporary Redirect`.
public const REDIRECT_TEMPORARY_REDIRECT_307 = 307;
# Represents the HTTP redirect status code `308 - Permanent Redirect`.
public const REDIRECT_PERMANENT_REDIRECT_308 = 308;
