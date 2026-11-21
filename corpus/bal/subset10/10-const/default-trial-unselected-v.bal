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

// W is defined first, so X's defaults resolve while W's still wait. X's default
// selects V from W|V; trying W must not resolve W's defaults, which construct X.
type V record {|
    int b;
|};

type W record {|
    string c = "w";
    X x = {};
|};

type X record {|
    int a = 1;
    W|V f = {b: 1};
|};

public function main() {
    X x = {};
    io:println(x); // @output {"a":1,"f":{"b":1}}
    W w = {};
    io:println(w); // @output {"c":"w","x":{"a":1,"f":{"b":1}}}
}
