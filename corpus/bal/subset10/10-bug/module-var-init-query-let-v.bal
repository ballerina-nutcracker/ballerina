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

type R record {
    int a;
    string b = "d";
};

function intOrErr() returns int|error {
    return 5;
}

final int[] doubled = from int i in [1, 2, 3]
    let int k = i * 2
    select k;

var inferred = 3;
int|string widened = 4;
float fl = 5;
decimal dc = 1.5;
byte bt = 255;
int:Signed8 s8 = -128;
final readonly & int[] ro = [1, 2];
map<int> mi = {a: 1};
R r = {a: 1};
int|error ie = 3;
string? ns = ();
int checked = check intOrErr();
final function () returns int fn = function() returns int {
    return mi.get("a") + 1;
};
[int, string] tup = [1, "x"];

public function main() {
    io:println(doubled); // @output [2,4,6]
    io:println(inferred); // @output 3
    io:println(widened); // @output 4
    io:println(fl, " ", dc, " ", bt, " ", s8); // @output 5.0 1.5 255 -128
    io:println(ro, " ", mi, " ", r); // @output [1,2] {"a":1} {"a":1,"b":"d"}
    io:println(ie, " ", ns, " ", checked, " ", fn(), " ", tup); // @output 3  5 2 [1,"x"]
}
