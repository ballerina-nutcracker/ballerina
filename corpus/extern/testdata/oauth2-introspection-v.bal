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

import ballerina/io;
import ballerina/lang.runtime;
import ballerina/oauth2;

const BASE = "http://testserver";

function show(oauth2:IntrospectionResponse|oauth2:Error result) {
    if result is oauth2:Error {
        error? cause = result.cause();
        io:println(result.message(), " | ", cause is error ? cause.message() : "no cause");
    } else {
        io:println(result.active, " ", result?.username ?: "-");
    }
}

public function main() returns error? {
    oauth2:ListenerOAuth2Provider provider = new ({
        url: BASE + "/introspect",
        tokenTypeHint: "access_token",
        optionalParams: {"extra": " v "},
        clientConfig: {auth: {tokenUrl: BASE + "/token", clientId: "c1", clientSecret: "s1"}}
    });
    oauth2:IntrospectionResponse active = check provider.authorize("active");
    io:println(active.active, " ", active?.scope, " ", active?.clientId, " ", active?.username, " ", active?.tokenType); // @output true read write c1 alice Bearer
    io:println(active?.iat, " ", active?.nbf, " ", active?.sub, " ", active?.aud, " ", active?.iss, " ", active?.jti); // @output 1 1 s a i j
    io:println(active["ext"]); // @output {"auth":"Bearer cc-1","extra":"v","hint":"access_token"}
    show(provider.authorize("revoked")); // @output false -
    show(provider.authorize("revoked", {"q": "1"})); // @output false -
    show(provider.authorize("")); // @output Credential cannot be empty. | no cause
    show(provider.authorize("numeric-string-exp")); // @output true -
    show(provider.authorize("no-exp")); // @output true bob

    // jBallerina panics on these responses: https://github.com/ballerina-platform/ballerina-library/issues/9716
    show(provider.authorize("string-exp")); // @output OAuth2 validation failed. | Invalid value for the introspection response field 'exp'.
    show(provider.authorize("decimal-exp")); // @output OAuth2 validation failed. | Invalid value for the introspection response field 'exp'.
    show(provider.authorize("aud-array")); // @output OAuth2 validation failed. | Invalid value for the introspection response field 'aud'.
    show(provider.authorize("active-string")); // @output OAuth2 validation failed. | Invalid value for the introspection response field 'active'.
    show(provider.authorize("iat-string")); // @output OAuth2 validation failed. | Invalid value for the introspection response field 'iat'.
    oauth2:ListenerOAuth2Provider arrayResponse = new ({url: BASE + "/introspect"});
    show(arrayResponse.authorize("array")); // @output OAuth2 validation failed. | The introspection response is not a JSON object.

    oauth2:ListenerOAuth2Provider failing = new ({url: BASE + "/401"});
    show(failing.authorize("x")); // @output OAuth2 validation failed. | Failed to call the introspection endpoint 'http://testserver/401'.
    oauth2:ListenerOAuth2Provider badJson = new ({url: BASE + "/bad-json"});
    show(badJson.authorize("x")); // @output OAuth2 validation failed. | Failed to convert 'not json' to JSON.
    oauth2:ListenerOAuth2Provider refreshAuth = new ({url: BASE + "/introspect",
        clientConfig: {auth: {refreshUrl: BASE + "/token", refreshToken: "r", clientId: "c", clientSecret: "s"}}});
    show(refreshAuth.authorize("revoked")); // @output false -

    // Cached responses are reused until their exp, or until defaultTokenExpTime when the response has no exp.
    oauth2:ListenerOAuth2Provider cached = new ({url: BASE + "/introspect", defaultTokenExpTime: 1,
        cacheConfig: {capacity: 10, evictionFactor: 0.25}});
    oauth2:IntrospectionResponse first = check cached.authorize("short");
    oauth2:IntrospectionResponse second = check cached.authorize("short");
    io:println(first?.exp == second?.exp); // @output true
    show(cached.authorize("no-exp")); // @output true bob
    show(cached.authorize("no-exp")); // @output true bob
    runtime:sleep(2);
    oauth2:IntrospectionResponse renewed = check cached.authorize("short");
    io:println(renewed?.exp != first?.exp); // @output true
    show(cached.authorize("no-exp")); // @output true bob
}
