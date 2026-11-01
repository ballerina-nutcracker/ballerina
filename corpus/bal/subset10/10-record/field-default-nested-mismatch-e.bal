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
    record {| int y = "a"; |} r; // @error nested default is not an int
    record {| int y = "b"; |}[] arr; // @error default nested in an array is not an int
    map<record {| int y = "c"; |}> m; // @error default nested in a map is not an int
    record {| int y = "d"; |}? u; // @error default nested in a union is not an int
    [record {| int y = "e"; |}] tup; // @error default nested in a tuple is not an int
    error<record {| int y = "g"; |}> err; // @error default nested in an error detail type is not an int
    readonly & record {| int y = "h"; |} ro; // @error default nested in an intersection is not an int
    int|record {| int y = "i"; |} ru; // @error default nested in the right operand of a union is not an int
    record {| int y = "j"; |} & readonly lro; // @error default nested in the left operand of an intersection is not an int
    [int, record {| int y = "k"; |}...] trest; // @error default nested in a tuple rest type is not an int
    record {| int y = "f"; |}...; // @error default nested in a rest type is not an int
|};

public function main() {
    T t = {r: {}, arr: [], m: {}, u: (), tup: [{}], err: error("x", y = 1), ro: {y: 1}, ru: 1, lro: {y: 1}, trest: [1]};
    io:println(t);
}
