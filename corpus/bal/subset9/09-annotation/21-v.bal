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

// Exercises the evaluateConstantReference !ok path: a non-const module
// variable has no singleton type, so constantSingleShapeValue returns false
// and the annotation falls back to a runtime annotation global.
type Code record {|
    int value;
|};

annotation Code codeAnnot on type;

int runtimeCode = 77;

@codeAnnot {value: runtimeCode}
type RuntimeAnnotTarget int;

public function main() {
    Code? c = RuntimeAnnotTarget.@codeAnnot;
    if c is Code {
        io:println(c.value); // @output 77
    }
}
