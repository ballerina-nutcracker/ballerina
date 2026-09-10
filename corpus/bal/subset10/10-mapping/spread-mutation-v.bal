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

type Counter record {|
    int p;
|};

function bump(Counter c) returns int {
    c.p = 9;
    return 0;
}

public function main() {
    Counter c = {p: 1};
    var built = {...c, q: bump(c)};
    // Operands are evaluated in source order and the spread source is expanded only when the
    // mapping is constructed, so the later mutation is visible in the result.
    io:println(built.p); // @output 9
    io:println(built.q); // @output 0
    io:println(c.p); // @output 9
}
