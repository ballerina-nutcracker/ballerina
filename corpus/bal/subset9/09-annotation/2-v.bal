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

type Info readonly & record {|
    string label;
|};

annotation Info runtimeInfo on type;
const annotation Info sourceInfo on source type;

@runtimeInfo {label: "runtime"}
@sourceInfo {label: "source"}
type Person record {|
    string name;
|};

public function main() {
    Info? runtimeValue = Person.@runtimeInfo;
    if runtimeValue is Info {
        io:println(runtimeValue.label); // @output runtime
    }

    Info? sourceValue = Person.@sourceInfo;
    io:println(sourceValue is ()); // @output true
}
