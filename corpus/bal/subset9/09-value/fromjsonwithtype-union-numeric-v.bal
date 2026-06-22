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

// Exercises the second loop in getConvertibleUnionMember where the value does
// not match any union member exactly but can be reached via numeric coercion.

type IntOrString int|string;
type OptFloat float?;
type DecimalOrString decimal|string;

public function main() returns error? {
    // float64 → int|string: float doesn't match int or string exactly in first
    // loop, but numeric coercion float→int succeeds in the second loop.
    json f = 2.0;
    IntOrString n = check f.fromJsonWithType(IntOrString);
    io:println(n); // @output 2

    // int64 → float?: int doesn't match float or () exactly in first loop,
    // but numeric coercion int→float succeeds in the second loop.
    json i = 3;
    OptFloat optFloat = check i.fromJsonWithType(OptFloat);
    io:println(optFloat); // @output 3.0

    // int64 → decimal|string: numeric coercion int→decimal via second loop.
    json j = 10;
    DecimalOrString ds = check j.fromJsonWithType(DecimalOrString);
    io:println(ds); // @output 10
    return;
}
