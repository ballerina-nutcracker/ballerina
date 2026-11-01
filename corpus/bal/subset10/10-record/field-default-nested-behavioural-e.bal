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

type T record {|
    function (record {| int y = "a"; |} r) f; // @error default nested in a parameter type is not an int
    function () returns record {| int y = "b"; |} g; // @error default nested in a return type is not an int
    function (record {| int y = "c"; |}... rs) h; // @error default nested in a rest parameter type is not an int
    stream<record {| int y = "d"; |}> s; // @error default nested in a stream type is not an int
    object { record {| int y = "e"; |} r; } o; // @error default nested in an object field type is not an int
    stream<int, error<record {| int y = "f"; |}>?> sc; // @error default nested in a stream completion type is not an int
    object { function m(record {| int y = "g"; |} r); } om; // @error default nested in an object method declaration is not an int
|};

public function main() {
    T? t = ();
    io:println(t is ());
}
