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

type Meta record {|
    string name;
|};

annotation Meta info on type;

@info {name: "myInt"}
type MyInt int;

// Built-in type literal as a typedesc — exercises resolveTypedescExpr.
function getIntType() returns typedesc<int> {
    return int;
}

// Unannotated type in typedesc position — covers the bir_gen
// annotationValuesForTypeSymbol path where the symbol has no annotations.
type Plain int;

function getPlainType() returns typedesc<Plain> {
    return Plain;
}

public function main() {
    // Built-in typedesc literal
    typedesc<int> intType = getIntType();
    io:println(intType is typedesc); // @output true

    // Unannotated type symbol in typedesc position
    typedesc<Plain> plainType = getPlainType();
    io:println(plainType is typedesc); // @output true

    // Annotation access on annotated type
    Meta? m = MyInt.@info;
    if m is Meta {
        io:println(m.name); // @output myInt
    }
}
