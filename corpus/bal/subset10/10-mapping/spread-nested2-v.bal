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

type Leaf record {|
    float value;
|};

type Middle record {|
    Leaf leaf;
    decimal weight;
|};

type Outer record {|
    Middle middle;
|};

type Pair record {|
    int a;
    int b;
|};

public function main() {
    // Literals several levels down still get the target's numeric context.
    Outer nested = {middle: {leaf: {value: 1}, weight: 2}};
    io:println(nested); // @output {"middle":{"leaf":{"value":1.0},"weight":2}}
    io:println(nested.middle.leaf.value is float); // @output true
    io:println(nested.middle.weight is decimal); // @output true

    // A nested constructor used as a spread operand keeps exactly its own keys.
    Pair p = {...{a: 1, b: 2}};
    io:println(p); // @output {"a":1,"b":2}
}
