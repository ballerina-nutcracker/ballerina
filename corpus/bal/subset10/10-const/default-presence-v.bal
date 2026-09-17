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

type Nils record {|
    int? a = ();
    int? b = 1;
|};

type Falsy record {|
    boolean b = false;
    int i = 0;
    float f = 0.0;
    string s = "";
    int[] l = [];
    map<int> m = {};
|};

type Optional record {|
    int x = 1;
    string s?;
|};

type Open record {
    int x = 1;
};

const Nils A = {};
const Nils B = {b: ()};
const Falsy C = {};
const Optional D = {};
const Open E = {"y": "extra"};

public function main() {
    io:println(A); // @output {"a":null,"b":1}
    io:println(B); // @output {"b":null,"a":null}
    io:println(C); // @output {"b":false,"i":0,"f":0.0,"s":"","l":[],"m":{}}
    io:println(C is readonly); // @output true
    io:println(D); // @output {"x":1}
    io:println(D.hasKey("s")); // @output false
    io:println(E); // @output {"y":"extra","x":1}
}
