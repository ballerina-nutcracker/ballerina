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
    record {|int x;|} closed = {...{x: 1}};
    io:println(closed); // @output {"x":1}

    record {|float x;|} contextual = {...{x: 1}};
    io:println(contextual); // @output {"x":1.0}

    record {|int x; int y;|} combined = {...{x: 1}, y: 2};
    io:println(combined); // @output {"x":1,"y":2}

    record {|int x; string y;|} both = {...{x: 1}, ...{y: "a"}};
    io:println(both); // @output {"x":1,"y":"a"}

    record {|int x;|} empty = {...{}, x: 3};
    io:println(empty); // @output {"x":3}
}
