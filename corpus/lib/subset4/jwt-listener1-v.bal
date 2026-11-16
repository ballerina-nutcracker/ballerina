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
import ballerina/jwt;
import ballerina/lang.runtime;

function show(jwt:Payload|jwt:Error result) {
    if result is jwt:Error {
        error? cause = result.cause();
        io:println(result.message(), " | ", cause is error ? cause.message() : "no cause");
    } else {
        io:println("authenticated ", result?.sub ?: "-", " ", result["n"] ?: "-");
    }
}

public function main() returns error? {
    jwt:ListenerJwtAuthProvider provider = new ({
        issuer: "wso2",
        signatureConfig: {secret: "secret"},
        cacheConfig: {capacity: 10, evictionFactor: 0.25, defaultMaxAge: -1}
    });
    string token = check jwt:issue({issuer: "wso2", username: "alice", customClaims: {"n": 1}, expTime: 1,
        signatureConfig: {algorithm: jwt:HS256, config: "secret"}});
    show(provider.authenticate(token)); // @output authenticated alice 1
    // The second call is answered from the JWT cache.
    show(provider.authenticate(token)); // @output authenticated alice 1
    show(provider.authenticate("a.b")); // @output Credential format does not match to JWT format. | no cause
    show(provider.authenticate("a.b.c.d")); // @output Credential format does not match to JWT format. | no cause
    show(provider.authenticate(token + ".")); // @output Credential format does not match to JWT format. | no cause
    string otherIssuer = check jwt:issue({issuer: "x", signatureConfig: {algorithm: jwt:HS256, config: "secret"}});
    show(provider.authenticate(otherIssuer)); // @output JWT validation failed. | JWT contained invalid issuer name 'x'
    // A cached token is rechecked for expiry and dropped from the cache once it has expired.
    runtime:sleep(2);
    show(provider.authenticate(token)); // @output JWT validation failed. | JWT is expired.

    jwt:ListenerJwtAuthProvider withoutCache = new ({username: "bob"});
    string bobToken = check jwt:issue({username: "bob", signatureConfig: {algorithm: jwt:HS512, config: "s"}});
    show(withoutCache.authenticate(bobToken)); // @output authenticated bob -
    show(withoutCache.authenticate(bobToken)); // @output authenticated bob -
    string unsigned = check jwt:issue({});
    show(withoutCache.authenticate(unsigned + ".x")); // @output JWT validation failed. | Unsupported signing algorithm 'none'.
}
