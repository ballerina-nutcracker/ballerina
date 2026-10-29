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

type A record {|
    map<int> v;
    int m;
|};

type B record {|
    map<string> v;
    int m;
|};

public function main() {
    [string, int][] pairs = [["a", 1], ["b", 2]];
    A|B r = {
        v: map from var p in pairs
            select [p[0], p[1] * 2],
        m: 1
    };
    io:println(r); // @output {"v":{"a":2,"b":4},"m":1}
    io:println(r is A); // @output true
}
