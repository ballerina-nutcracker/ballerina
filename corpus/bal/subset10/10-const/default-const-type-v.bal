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

// H's field type is the constant K, evaluated while H is being defined and
// S's defaults are still waiting: K's shape includes S's default a.
type H record {|
    S s;
    K k;
|};

type S record {|
    int a = 1;
|};

const S K = {};

public function main() {
    H h = {s: {}, k: {a: 1}};
    io:println(h); // @output {"s":{"a":1},"k":{"a":1}}
    io:println(K); // @output {"a":1}
}
