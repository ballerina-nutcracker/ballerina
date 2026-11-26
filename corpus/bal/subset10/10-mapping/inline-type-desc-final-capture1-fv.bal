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

type A record {|
    boolean x;
|};

type B record {|
    string x;
|};

public function main() {
    final int k = 5;
    record {| int a; |} value = {a: 2};
    any v = value;
    A|B tested = {x: v is record {| int a = k; |}}; // @error inline type with a default in a candidate trial is unimplemented
    A|B cast = {x: (<record {| int a = k; |}>v).a == 2}; // @error inline type with a default in a candidate trial is unimplemented
    A|B param = {x: v is function (int p = k)}; // @error inline type with a default in a candidate trial is unimplemented
    io:println(tested);
    io:println(cast);
    io:println(param);
}
