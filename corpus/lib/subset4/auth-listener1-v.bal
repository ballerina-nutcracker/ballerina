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

isolated class TokenProvider {
    *auth:ListenerBasicAuthProvider;

    public isolated function authenticate(string credential) returns auth:UserDetails|auth:Error {
        if credential == "ok" {
            return {username: "bob", scopes: ["admin", "dev"]};
        }
        return error auth:Error("custom failure");
    }
}

function show(auth:UserDetails|auth:Error result) {
    if result is auth:Error {
        io:println(result.message());
    } else {
        io:println(result.username, " ", result?.scopes);
    }
}

public function main() {
    auth:ListenerBasicAuthProvider custom = new TokenProvider();
    show(custom.authenticate("ok")); // @output bob ["admin","dev"]
    show(custom.authenticate("no")); // @output custom failure

    auth:UserDetails noScopes = {username: "carol"};
    io:println(noScopes?.scopes is ()); // @output true

    error plain = error("plain");
    io:println(plain is auth:Error); // @output false

    // Config.toml users are not supported yet (#781, #294), so the file user store is always unconfigured.
    auth:ListenerFileUserStoreBasicAuthProvider store = new;
    show(store.authenticate("")); // @output Credential cannot be empty.
    show(store.authenticate("YWxpY2U6czNjcmV0")); // @output File user store values are not provided or not properly configured.
    auth:ListenerBasicAuthProvider fromConfig = new auth:ListenerFileUserStoreBasicAuthProvider({});
    show(fromConfig.authenticate("!!!")); // @output File user store values are not provided or not properly configured.
}
