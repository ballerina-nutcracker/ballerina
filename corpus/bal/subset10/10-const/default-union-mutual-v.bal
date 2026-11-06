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

// A selects B as a field type, so B's defaults resolve while A's are still
// waiting; B's default selects A from a union and relies on A's default x.
type A record {|
    B? b = ();
    int x = 1;
|};

type B record {|
    (A|C)? v = {b: ()};
|};

type C record {|
    string s;
|};

const B K = {};

public function main() {
    B r = {};
    io:println(r); // @output {"v":{"b":null,"x":1}}
    io:println(K); // @output {"v":{"b":null,"x":1}}
}
