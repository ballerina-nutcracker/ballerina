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

// 'x' is excluded, so no value of Excluded holds it, but the rest still admits other keys.
type Excluded record {|
    never x?;
    int...;
|};

type OtherRest record {|
    string...;
|};

public function main() {
    Excluded src = {"y": 1};
    var kept = {...src};
    io:println(kept is record {| never x?; int...; |}); // @output true
    io:println(kept is map<int>); // @output true
    io:println(kept); // @output {"y":1}

    // A member that excludes a name does not stop another member from supplying it.
    var before = {x: "hello", ...src};
    io:println(before); // @output {"x":"hello","y":1}
    io:println(before is record {| string x; int...; |}); // @output true

    var after = {...src, x: "hello"};
    io:println(after); // @output {"y":1,"x":"hello"}
    io:println(after is record {| string x; int...; |}); // @output true

    // Another spread's rest may supply the name the first spread excludes, so the combined
    // shape no longer excludes it.
    record {| never y?; |} onlyExcludes = {};
    OtherRest other = {"x": "a"};
    var combined = {...onlyExcludes, ...other};
    io:println(combined is record {| never y?; string...; |}); // @output false
    io:println(combined is map<string>); // @output true
    io:println(combined); // @output {"x":"a"}
}
