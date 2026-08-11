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

public function main() {
    (record {| string kind; int first = 11; |}|record {| int code; int second = 12; |}) first = {kind: "one", first: 11};
    if first is record {| string kind; int first; |} {
        io:println(first.first); // @output 11
    }

    (record {| string kind; int first = 11; |}|record {| int code; int second = 12; |}) second = {code: 2, second: 12};
    if second is record {| int code; int second; |} {
        io:println(second.second); // @output 12
    }
}
