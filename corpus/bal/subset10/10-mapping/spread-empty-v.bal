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

type EmptyClosed record {|
|};

type NeverX record {|
    never x?;
|};

type NeverXOpen record {|
    never x?;
    int...;
|};

public function main() {
    EmptyClosed empty = {};
    var a = {x: 1, ...empty};
    io:println(a); // @output {"x":1}

    NeverX excluded = {};
    var b = {x: 2, ...excluded};
    io:println(b); // @output {"x":2}

    NeverXOpen open = {"y": 3};
    var c = {x: 4, ...open};
    io:println(c); // @output {"x":4,"y":3}

    map<never> nothing = {};
    var d = {x: 5, ...nothing};
    io:println(d); // @output {"x":5}

    NeverX one = {};
    NeverX two = {};
    var e = {...one, ...two};
    io:println(e); // @output {}
}
