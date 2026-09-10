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

// 'e' has an uninhabited type, so no value of Src can contain the key 'e'. An excluded member
// collides with nothing, even though its type is not the literal 'never'.
type Src record {|
    record {| never n; |} e?;
    int y;
|};

type Tgt record {|
    string e;
    int y;
|};

public function main() {
    Src s = {y: 1};
    Tgt a = {e: "hi", ...s};
    io:println(a); // @output {"e":"hi","y":1}

    Tgt b = {...s, e: "bye"};
    io:println(b); // @output {"y":1,"e":"bye"}

    record {| never x?; |} litEmpty = {};
    var c = {x: 1, ...litEmpty};
    io:println(c); // @output {"x":1}

    record {| never x?; int...; |} litOpen = {"y": 2};
    var d = {x: 1, ...litOpen};
    io:println(d); // @output {"x":1,"y":2}
}
