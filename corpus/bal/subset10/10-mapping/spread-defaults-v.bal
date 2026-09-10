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

isolated function tracedDefault() returns int {
    io:println("default evaluated");
    return 7;
}

type WithDefault record {|
    int x;
    int y = tracedDefault();
|};

type OptionalMember record {|
    int x;
    int y?;
|};

type SourceX record {|
    int x;
|};

type SourceXY record {|
    int x;
    int y;
|};

public function main() {
    SourceX only = {x: 1};
    WithDefault filled = {...only};
    // @output default evaluated
    io:println(filled); // @output {"x":1,"y":7}

    SourceXY complete = {x: 2, y: 9};
    WithDefault supplied = {...complete};
    io:println(supplied); // @output {"x":2,"y":9}

    OptionalMember present = {x: 3, y: 4};
    WithDefault fromOptional = {...present};
    io:println(fromOptional); // @output {"x":3,"y":4}

    OptionalMember absent = {x: 5};
    WithDefault defaulted = {...absent};
    // @output default evaluated
    io:println(defaulted); // @output {"x":5,"y":7}
}
