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

import ballerina/auth;
import ballerina/io;

function show(string|auth:Error result) {
    io:println(result is auth:Error ? result.message() : result);
}

public function main() returns error? {
    auth:ClientBasicAuthProvider provider = new ({username: "alice", password: "s3cret"});
    string token = check provider.generateToken();
    io:println(token); // @output YWxpY2U6czNjcmV0
    [string, string] credentials = check auth:extractUsernameAndPassword(token);
    io:println(credentials[0], " ", credentials[1]); // @output alice s3cret

    auth:CredentialsConfig config = {username: "élan", password: "päss"};
    auth:ClientBasicAuthProvider unicode = new (config);
    // Mutating the caller's record after init does not change the provider's credentials.
    config.password = "changed";
    show(unicode.generateToken()); // @output w6lsYW46cMOkc3M=

    show((new auth:ClientBasicAuthProvider({username: "alice", password: ""})).generateToken()); // @output Username or password cannot be empty.
    show((new auth:ClientBasicAuthProvider({username: "", password: "s3cret"})).generateToken()); // @output Username or password cannot be empty.
}
