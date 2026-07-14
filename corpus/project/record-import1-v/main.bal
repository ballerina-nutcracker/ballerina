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

import testorg/record_import1_v.types1;
import testorg/record_import1_v.types2;

type R record {|
    int? intField;
    float? floatField;
|};

public function main() {
    types1:R1 r1 = {intField: 255, floatField: 1.5};
    types2:R2 r2 = r1;
    r1 = r2;
    R r = r2;
    io:println(r is types1:R1); // @output true
    io:println(r is types2:R2); // @output true
    r = {intField: 17, floatField: 2.5};
    io:println(r is types1:R1); // @output false
    io:println(r is types2:R2); // @output false

    any v = types1:create(11, 3.5);
    io:println(types2:test(v)); // @output true
    io:println(types1:test(v)); // @output true
    io:println(types2:test(r)); // @output false

    v = types2:create(21, -3.5);
    io:println(types1:test(v)); // @output true
    io:println(types2:test(v)); // @output true
    io:println(types1:test(r)); // @output false
}
