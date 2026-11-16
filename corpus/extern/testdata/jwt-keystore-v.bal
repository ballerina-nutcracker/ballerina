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

import ballerina/crypto;
import ballerina/io;
import ballerina/jwt;

public function main() returns error? {
    crypto:KeyStore keyStore = {path: "testdata/crypto-keystore.p12", password: "secret"};
    string token = check jwt:issue({
        issuer: "wso2",
        signatureConfig: {algorithm: jwt:RS256, config: {keyStore, keyAlias: "ballerina", keyPassword: "secret"}}
    });
    int? dot = token.indexOf(".");
    io:println(token.substring(0, <int>dot)); // @output eyJhbGciOiJSUzI1NiIsICJ0eXAiOiJKV1QifQ
    io:println((check jwt:decode(token))[1].iss); // @output wso2

    string|jwt:Error wrongPassword = jwt:issue({
        signatureConfig: {config: {keyStore: {path: "testdata/crypto-keystore.p12", password: "wrong"}, keyAlias: "ballerina", keyPassword: "wrong"}}
    });
    io:println(wrongPassword is jwt:Error ? wrongPassword.message() : "issued"); // @output Failed to decode private key.

    jwt:ClientSelfSignedJwtAuthProvider provider = new ({
        username: "alice",
        signatureConfig: {algorithm: jwt:RS512, config: {keyStore, keyAlias: "ballerina", keyPassword: "secret"}}
    });
    io:println((check jwt:decode(check provider.generateToken()))[1].sub); // @output alice
}
