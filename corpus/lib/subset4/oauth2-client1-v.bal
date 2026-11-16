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
import ballerina/oauth2;

function initFailure(oauth2:GrantConfig config) {
    oauth2:ClientOAuth2Provider|error provider = trap new (config);
    if provider is error {
        error? cause = provider.cause();
        io:println(provider is oauth2:Error, " ", provider.message(), " | ", cause is error ? cause.message() : "no cause");
    } else {
        io:println("created");
    }
}

public function main() {
    io:println(oauth2:DEFAULT_CONNECT_TIMEOUT, " ", oauth2:DEFAULT_REQ_TIMEOUT); // @output 15 30
    io:println(oauth2:globalConnectTimeout, " ", oauth2:globalReqTimeout); // @output 15 30
    io:println(oauth2:HTTP_1_1, " ", oauth2:HTTP_2, " ", oauth2:AUTH_HEADER_BEARER, " ", oauth2:POST_BODY_BEARER, " ", oauth2:INFER_REFRESH_CONFIG); // @output HTTP_1_1 HTTP_2 AUTH_HEADER_BEARER POST_BODY_BEARER INFER_REFRESH_CONFIG

    oauth2:ClientCredentialsGrantConfig config = {tokenUrl: "https://example.com/token", clientId: "c", clientSecret: "s"};
    io:println(config.defaultTokenExpTime, " ", config.clockSkew, " ", config.credentialBearer, " ", config.clientConfig); // @output 3600 0 AUTH_HEADER_BEARER {"httpVersion":"HTTP_1_1"}

    initFailure({tokenUrl: "https://example.com/token", clientId: "", clientSecret: "s"}); // @output true Client-id or client-secret cannot be empty. | no cause
    initFailure({refreshUrl: "https://example.com/token", refreshToken: "r", clientId: "c", clientSecret: ""}); // @output true Client-id or client-secret cannot be empty. | no cause
    // The test platform answers every request with "test body", which is not JSON.
    initFailure(config); // @output true Failed to get JSON from the response payload. | invalid character 'e' in literal true (expecting 'r')
    initFailure({tokenUrl: "https://example.com/token", username: "u", password: "p"}); // @output true Failed to get JSON from the response payload. | invalid character 'e' in literal true (expecting 'r')
    initFailure({tokenUrl: "https://example.com/token", assertion: "a.b.c"}); // @output true Failed to get JSON from the response payload. | invalid character 'e' in literal true (expecting 'r')
}
