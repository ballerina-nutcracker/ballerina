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

// A mapping-valued annotation gets the record's constant field defaults filled
// in, while a non-const module variable still falls back to a runtime global.
type Code record {|
    int value = 5;
    string label = "none";
|};

annotation Code codeAnnot on type;

int runtimeCode = 77;

@codeAnnot {}
type FoldedTarget int;

@codeAnnot {value: runtimeCode}
type RuntimeTarget int;

public function main() {
    Code? folded = FoldedTarget.@codeAnnot;
    if folded is Code {
        io:println(folded); // @output {"value":5,"label":"none"}
    }
    Code? runtimeValue = RuntimeTarget.@codeAnnot;
    if runtimeValue is Code {
        io:println(runtimeValue); // @output {"value":77,"label":"none"}
    }
}
