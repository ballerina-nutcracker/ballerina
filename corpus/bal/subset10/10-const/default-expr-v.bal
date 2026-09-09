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

const int BASE = 10;
const string NAME = "bal";

type Exprs record {|
    int sum = BASE + 5;
    int neg = -BASE;
    float conv = <float>BASE;
    string tmpl = string `hi ${NAME}`;
    int cond = BASE > 5 ? 1 : 2;
    int guard = BASE > 5 ? 1 : 1 / 0;
    int[] list = [BASE, 2];
    map<string> mapping = {k: NAME};
    Inner inner = {};
|};

type Inner record {|
    int n = BASE * 2;
|};

const Exprs A = {};

public function main() {
    io:println(A.sum); // @output 15
    io:println(A.neg); // @output -10
    io:println(A.conv); // @output 10.0
    io:println(A.tmpl); // @output hi bal
    io:println(A.cond); // @output 1
    io:println(A.guard); // @output 1
    io:println(A.list); // @output [10,2]
    io:println(A.mapping); // @output {"k":"bal"}
    io:println(A.inner.n); // @output 20
    io:println(A is readonly); // @output true
}
