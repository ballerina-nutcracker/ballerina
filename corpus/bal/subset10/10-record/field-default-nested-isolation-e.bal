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

int counter = 3;

function f() returns int => 3;

type T record {|
    record {| int y = counter; |} r; // @error nested default reads a mutable module variable
    record {| int y = counter; |}[] arr; // @error default nested in an array reads a mutable module variable
    map<record {| int y = f(); |}> m; // @error default nested in a map calls a non-isolated function
    [record {| int y = f(); |}] tup; // @error default nested in a tuple calls a non-isolated function
    record {| int y = counter; |}...; // @error default nested in a rest type reads a mutable module variable
|};

public function main() {
    T t = {r: {y: 1}, arr: [], m: {}, tup: [{y: 1}]};
    io:println(t);
}
