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
    boolean x;
|};

type B record {|
    string x;
|};

public function main() {
    final int k = 5;
    record {| int a; |} value = {a: 2};
    any v = value;
    A|B tested = {x: v is record {| int a = k; |}};
    A|B cast = {x: (<record {| int a = k; |}>v).a == 2};
    A|B param = {x: v is function (int p = k)};
    io:println(tested); // @output {"x":true}
    io:println(cast); // @output {"x":true}
    io:println(param); // @output {"x":false}
}
