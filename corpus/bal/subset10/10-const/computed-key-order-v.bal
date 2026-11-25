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

// The inherent type is map<int>, so the computed fields are added after the
// other fields, in source order.
const map<int> M1 = {["c"]: 1, ["b"]: 2, "a": 3};
// Without a contextually expected type, computed fields are also added last.
const M2 = {["a"]: 1, "b": 2, ["c"]: 3};

public function main() {
    io:println(M1); // @output {"a":3,"c":1,"b":2}
    io:println(M2); // @output {"b":2,"a":1,"c":3}
}
