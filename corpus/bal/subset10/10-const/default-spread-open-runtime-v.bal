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

isolated function next() returns int {
    return 42;
}

type Runtime record {
    int y = next();
};

const map<int> Z = {x: 1};

// An open spread operand may supply any field, but Z does not supply y, so r
// cannot be folded and is evaluated at runtime.
type R record {|
    Runtime r = {...<map<int>>Z};
|};

public function main() {
    R v = {};
    io:println(v); // @output {"r":{"x":1,"y":42}}
}
