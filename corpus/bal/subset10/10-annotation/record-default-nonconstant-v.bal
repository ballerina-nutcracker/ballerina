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

// A record default the evaluator cannot fold must not turn a permitted
// non-const annotation value into an error: it falls back to a runtime value.
isolated function seed() returns int {
    return 11;
}

type Code record {|
    int value = seed();
    string label = "none";
|};

annotation Code codeAnnot on type;

@codeAnnot {}
type RuntimeDefaultTarget int;

public function main() {
    Code? c = RuntimeDefaultTarget.@codeAnnot;
    if c is Code {
        io:println(c); // @output {"value":11,"label":"none"}
    }
}
