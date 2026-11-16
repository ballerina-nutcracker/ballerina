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

function show(string|oauth2:Error result) {
    if result is oauth2:Error {
        error? cause = result.cause();
        error? rootCause = cause is error ? cause.cause() : ();
        io:println(result.message(), " | ", cause is error ? cause.message() : "no cause", " | ",
            rootCause is error ? rootCause.message() : "no cause");
    } else {
        io:println(result);
    }
}

function provider(oauth2:GrantConfig config) returns oauth2:ClientOAuth2Provider|error {
    oauth2:ClientOAuth2Provider|error result = trap new (config);
    if result is error {
        error? cause = result.cause();
        io:println("init failed: ", result.message(), " | ", cause is error ? cause.message() : "no cause");
    }
    return result;
}

function initFails(oauth2:GrantConfig config) {
    oauth2:ClientOAuth2Provider|error result = provider(config);
    if result is oauth2:ClientOAuth2Provider {
        io:println("created");
    }
}

public function main() returns error? {
    oauth2:ClientOAuth2Provider clientCredentials = check provider({tokenUrl: BASE + "/token", clientId: "c1", clientSecret: "s1", scopes: ["read", " write ", ""]});
    show(clientCredentials.generateToken()); // @output cc-1
    show(clientCredentials.generateToken()); // @output cc-1

    oauth2:ClientOAuth2Provider headerBearer = check provider({tokenUrl: BASE + "/echo", clientId: "c 1", clientSecret: "s&1", scopes: "  a b  ",
        optionalParams: {" k ": " v "}, clientConfig: {customHeaders: {"X-Custom": "h1", "Content-Type": "text/plain"}, customPayload: "extra=1"}});
    show(headerBearer.generateToken()); // @output {"auth":"Basic YyAxOnMmMQ==","body":"grant_type=client_credentials&scope=a b&k=v&extra=1","ct":"application/x-www-form-urlencoded","x":["h1"]}
    oauth2:ClientOAuth2Provider bodyBearer = check provider({tokenUrl: BASE + "/echo", clientId: "c1", clientSecret: "s1", credentialBearer: oauth2:POST_BODY_BEARER});
    show(bodyBearer.generateToken()); // @output {"auth":"","body":"grant_type=client_credentials&client_id=c1&client_secret=s1","ct":"application/x-www-form-urlencoded","x":null}
    oauth2:ClientOAuth2Provider password = check provider({tokenUrl: BASE + "/echo", username: "al ice", password: "p@ss&word", clientId: "c1", clientSecret: "s1"});
    show(password.generateToken()); // @output {"auth":"Basic YzE6czE=","body":"grant_type=password&username=al%20ice&password=p%40ss%26word","ct":"application/x-www-form-urlencoded","x":null}
    oauth2:ClientOAuth2Provider jwtBearer = check provider({tokenUrl: BASE + "/echo", assertion: "a.b.c"});
    show(jwtBearer.generateToken()); // @output {"auth":"","body":"grant_type=urn:ietf:params:oauth:grant-type:jwt-bearer&assertion=a.b.c","ct":"application/x-www-form-urlencoded","x":null}
    oauth2:ClientOAuth2Provider refreshToken = check provider({refreshUrl: BASE + "/echo", refreshToken: "r1", clientId: "c1", clientSecret: "s1", scopes: ["x"]});
    show(refreshToken.generateToken()); // @output {"auth":"Basic YzE6czE=","body":"grant_type=refresh_token&refresh_token=r1&scope=x","ct":"application/x-www-form-urlencoded","x":null}

    // Tokens from /token-short expire after a second, so the next call refreshes them.
    oauth2:ClientOAuth2Provider passwordRefresh = check provider({tokenUrl: BASE + "/token-short", username: "u", password: "p", clientId: "c1", clientSecret: "s1", refreshConfig: oauth2:INFER_REFRESH_CONFIG});
    oauth2:ClientOAuth2Provider passwordNoRefresh = check provider({tokenUrl: BASE + "/token-short", username: "u", password: "p", clientId: "c1", clientSecret: "s1"});
    oauth2:ClientOAuth2Provider passwordNoClient = check provider({tokenUrl: BASE + "/token-short", username: "u", password: "p", refreshConfig: oauth2:INFER_REFRESH_CONFIG});
    oauth2:ClientOAuth2Provider passwordExplicitRefresh = check provider({tokenUrl: BASE + "/token-short", username: "u", password: "p", clientId: "c1", clientSecret: "s1",
        optionalParams: {"p": "1"}, refreshConfig: {refreshUrl: BASE + "/echo", scopes: "s", optionalParams: {"q": "2"}, credentialBearer: oauth2:POST_BODY_BEARER}});
    oauth2:ClientOAuth2Provider passwordInferredParams = check provider({tokenUrl: BASE + "/token-short", username: "u", password: "p", clientId: "c1", clientSecret: "s1",
        optionalParams: {"p": "1"}, refreshConfig: oauth2:INFER_REFRESH_CONFIG});
    oauth2:ClientOAuth2Provider refreshGrant = check provider({refreshUrl: BASE + "/token-short", refreshToken: "initial", clientId: "c1", clientSecret: "s1"});
    oauth2:ClientOAuth2Provider jwtBearerRefresh = check provider({tokenUrl: BASE + "/token-short", assertion: "a.b.c", clientId: "c1", clientSecret: "s1"});
    oauth2:ClientOAuth2Provider jwtBearerNoClient = check provider({tokenUrl: BASE + "/token-short", assertion: "a.b.c"});
    oauth2:ClientOAuth2Provider clientCredentialsShort = check provider({tokenUrl: BASE + "/token-short", clientId: "c1", clientSecret: "s1"});
    oauth2:ClientOAuth2Provider defaultExpiry = check provider({tokenUrl: BASE + "/no-expiry", clientId: "c1", clientSecret: "s1", defaultTokenExpTime: 1});
    // A clock skew larger than the token lifetime makes every call request a new token.
    oauth2:ClientOAuth2Provider skewed = check provider({tokenUrl: BASE + "/token", clientId: "c1", clientSecret: "s1", clockSkew: 4000});
    show(skewed.generateToken()); // @output cc-18
    show(skewed.generateToken()); // @output cc-19
    runtime:sleep(2);
    show(passwordRefresh.generateToken()); // @output refreshed-rt-7
    show(passwordRefresh.generateToken()); // @output refreshed-rt-7
    show(passwordNoRefresh.generateToken()); // @output Failed to generate OAuth2 token. | Failed to refresh access token since refresh configurations are not provided. | no cause
    show(passwordNoClient.generateToken()); // @output Failed to generate OAuth2 token. | Client-id or client-secret cannot be empty. | no cause
    show(passwordExplicitRefresh.generateToken()); // @output {"auth":"","body":"grant_type=refresh_token&refresh_token=rt-10&scope=s&q=2&client_id=c1&client_secret=s1","ct":"application/x-www-form-urlencoded","x":null}
    show(passwordInferredParams.generateToken()); // @output refreshed-rt-11
    show(refreshGrant.generateToken()); // @output refreshed-rt-r12
    show(jwtBearerRefresh.generateToken()); // @output refreshed-rt-13
    show(jwtBearerNoClient.generateToken()); // @output Failed to generate OAuth2 token. | Client-id or client-secret cannot be empty. | no cause
    show(clientCredentialsShort.generateToken()); // @output cc-25
    show(defaultExpiry.generateToken()); // @output ne-26

    initFails({tokenUrl: BASE + "/token", clientId: "", clientSecret: "s"}); // @output init failed: Client-id or client-secret cannot be empty. | no cause
    initFails({tokenUrl: BASE + "/token", username: "u", password: "p", clientId: "", clientSecret: "s"}); // @output init failed: Client-id or client-secret cannot be empty. | no cause
    initFails({refreshUrl: BASE + "/token", refreshToken: "r", clientId: "c", clientSecret: ""}); // @output init failed: Client-id or client-secret cannot be empty. | no cause
    initFails({tokenUrl: BASE + "/token", assertion: "a", clientId: "", clientSecret: "s"}); // @output init failed: Client-id or client-secret cannot be empty. | no cause
    initFails({tokenUrl: BASE + "/401", clientId: "c", clientSecret: "s"}); // @output init failed: Failed to call the token endpoint 'http://testserver/401'. | Failed to get a success response from the endpoint. Response code: '401'. Response body: 'unauthorized'
    initFails({tokenUrl: BASE + "/bad-json", clientId: "c", clientSecret: "s"}); // @output init failed: Failed to get JSON from the response payload. | invalid character 'o' in literal null (expecting 'u')
    initFails({tokenUrl: BASE + "/no-access-token", clientId: "c", clientSecret: "s"}); // @output init failed: Failed to access 'access_token' property from the JSON. | missing map key 'access_token'
    initFails({tokenUrl: BASE + "/number-access-token", clientId: "c", clientSecret: "s"}); // @output init failed: Failed to extract 'access_token' property as a 'string' from the JSON. | no cause
    initFails({tokenUrl: BASE + "/big-number", clientId: "c", clientSecret: "s"}); // @output init failed: Failed to get JSON from the response payload. | number '12345678901234567890' is out of the int range
    oauth2:ClientOAuth2Provider created = check provider({tokenUrl: BASE + "/created", clientId: "c", clientSecret: "s"});
    show(created.generateToken()); // @output cc-32
    initFails({tokenUrl: BASE + "/a b", clientId: "c", clientSecret: "s"}); // @output init failed: Failed to call the token endpoint 'http://testserver/a b'. | Failed to create URI for the provided value "http://testserver/a b".
    initFails({tokenUrl: BASE + "/token", clientId: "c", clientSecret: "s", clientConfig: {connectTimeout: 0}}); // @output init failed: Failed to call the token endpoint 'http://testserver/token'. | OAuth2 connection timeout must be greater than zero
    initFails({tokenUrl: BASE + "/token", clientId: "c", clientSecret: "s", clientConfig: {reqTimeout: -1}}); // @output init failed: Failed to call the token endpoint 'http://testserver/token'. | OAuth2 request timeout must be greater than zero
    oauth2:ClientOAuth2Provider|error hung = trap new ({tokenUrl: BASE + "/hang", clientId: "c", clientSecret: "s", clientConfig: {reqTimeout: 0.2}});
    io:println(hung is error ? hung.message() : "created"); // @output Failed to call the token endpoint 'http://testserver/hang'.
    oauth2:ClientOAuth2Provider noScheme = check provider({tokenUrl: "testserver/token", clientId: "c", clientSecret: "s", clientConfig: {httpVersion: oauth2:HTTP_2}});
    show(noScheme.generateToken()); // @output cc-34
    // jBallerina panics with IllegalArgumentException for these: https://github.com/ballerina-platform/ballerina-library/issues/9715
    initFails({tokenUrl: BASE + "/token", clientId: "c", clientSecret: "s", clientConfig: {customHeaders: {"Host": "x"}}}); // @output init failed: Failed to call the token endpoint 'http://testserver/token'. | Failed to send the request to the endpoint. restricted header name: "Host"
    initFails({tokenUrl: BASE + "/token", clientId: "c", clientSecret: "s", clientConfig: {customHeaders: {"content-length": "5"}}}); // @output init failed: Failed to call the token endpoint 'http://testserver/token'. | Failed to send the request to the endpoint. restricted header name: "content-length"
    io:println(oauth2:globalConnectTimeout, " ", oauth2:globalReqTimeout, " ", oauth2:DEFAULT_CONNECT_TIMEOUT, " ", oauth2:DEFAULT_REQ_TIMEOUT); // @output 15 30 15 30
}
