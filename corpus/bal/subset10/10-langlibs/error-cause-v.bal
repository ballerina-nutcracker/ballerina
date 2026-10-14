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

import ballerina/io;

type AppError distinct error;

public function main() {
    error root = error("root");
    io:println(root.cause() is ()); // @output true

    error mid = error("mid", root);
    error top = error("top", mid);

    io:println(error:message(top)); // @output top

    error? cause = error:cause(top);
    if cause is error {
        io:println(cause.message()); // @output mid
        error? inner = cause.cause();
        if inner is error {
            io:println(inner.message()); // @output root
        }
    }

    AppError app = error AppError("app", top);
    error? appCause = app.cause();
    io:println(appCause is error ? appCause.message() : "none"); // @output top
}
