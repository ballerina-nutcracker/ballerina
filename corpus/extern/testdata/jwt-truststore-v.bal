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
    crypto:TrustStore trustStore = {path: "testdata/crypto-keystore.p12", password: "secret"};
    string token = check jwt:issue({
        username: "alice",
        signatureConfig: {algorithm: jwt:RS384, config: {keyStore, keyAlias: "ballerina", keyPassword: "secret"}}
    });
    jwt:Payload payload = check jwt:validate(token, {signatureConfig: {trustStoreConfig: {trustStore, certAlias: "ballerina"}}});
    io:println(payload.sub); // @output alice

    string hmacToken = check jwt:issue({signatureConfig: {algorithm: jwt:HS256, config: "secret"}});
    jwt:Payload|jwt:Error wrongAlgorithm = jwt:validate(hmacToken, {signatureConfig: {trustStoreConfig: {trustStore, certAlias: "ballerina"}}});
    io:println(wrongAlgorithm is jwt:Error ? wrongAlgorithm.message() : "valid"); // @output Unsupported RSA algorithm 'HS256'.

    jwt:Payload|jwt:Error missingStore = jwt:validate(token, {signatureConfig: {trustStoreConfig: {trustStore: {path: "testdata/missing.p12", password: "secret"}, certAlias: "ballerina"}}});
    io:println(missingStore is jwt:Error ? missingStore.message() : "valid"); // @output Failed to decode public key.

    jwt:ListenerJwtAuthProvider provider = new ({signatureConfig: {trustStoreConfig: {trustStore, certAlias: "ballerina"}}});
    io:println((check provider.authenticate(token)).sub); // @output alice
}
