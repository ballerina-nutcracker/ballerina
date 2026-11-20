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

function show([string, string]|auth:Error result) {
    if result is auth:Error {
        io:println(result.message(), " (cause: ", result.cause() is error, ")");
    } else {
        io:println(result[0], "|", result[1]);
    }
}

public function main() {
    show(auth:extractUsernameAndPassword("YWxpY2U6czNjcmV0")); // @output alice|s3cret
    // Only the first colon separates the username from the password.
    show(auth:extractUsernameAndPassword("YWxpY2U6czM6Y3JldA==")); // @output alice|s3:cret
    show(auth:extractUsernameAndPassword("w6lsw6huOnDDpHNz")); // @output élèn|päss

    show(auth:extractUsernameAndPassword("YWxpY2U6")); // @output Incorrect credential format. Format should be username:password (cause: false)
    show(auth:extractUsernameAndPassword("OnMzY3JldA==")); // @output Incorrect credential format. Format should be username:password (cause: false)
    show(auth:extractUsernameAndPassword("YWxpY2U=")); // @output Incorrect credential format. Format should be username:password (cause: false)
    show(auth:extractUsernameAndPassword("")); // @output Incorrect credential format. Format should be username:password (cause: false)
    show(auth:extractUsernameAndPassword("!!!")); // @output Failed to convert string credential to byte[]. (cause: true)
    show(auth:extractUsernameAndPassword("/w==")); // @output Failed to convert byte[] credential to string. (cause: true)
}
