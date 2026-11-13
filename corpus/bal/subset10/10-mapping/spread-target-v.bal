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

type Target record {|
    int x;
    string y;
|};

type WideRest record {|
    int|string...;
|};

type IntRest record {|
    int...;
|};

type ClosedWithRest record {|
    int x;
    never...;
|};

type SourceX record {|
    int x;
|};

public function main() {
    SourceX src = {x: 1};
    Target target = {...src, y: "hi"};
    io:println(target); // @output {"x":1,"y":"hi"}

    IntRest ints = {"a": 1};
    WideRest wide = {...ints};
    io:println(wide); // @output {"a":1}

    ClosedWithRest closed = {...src};
    io:println(closed); // @output {"x":1}

    map<never> nothing = {};
    Target fromNever = {...nothing, x: 2, y: "z"};
    io:println(fromNever); // @output {"x":2,"y":"z"}

    var withContext = {...src, nested: {a: 1.0}};
    io:println(withContext.nested); // @output {"a":1.0}
}
