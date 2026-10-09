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

function str() returns string {
    return "s";
}

function strLen(string s) returns int {
    return s.length();
}

function strOrErr() returns string|error {
    return "s";
}

float f = 1.5;
boolean flag = true;
int? opt = ();
final int a = "s"; // @error
int b = 1.50f; // @error
final byte c = 300; // @error
final int d = f; // @error
final string g = 1; // @error
isolated int h = true; // @error
int i = str(); // @error
int j = check strOrErr(); // @error
int k = flag ? 1 : "s"; // @error
int l = opt ?: "z"; // @error
int? m = 1.5; // @error
string|int n = 2.0; // @error
final int[] xs = from int x in [1, 2] select x.toString(); // @error
int q = from int x in [1] select x; // @error
final function (int) returns int fp = strLen; // @error

public function main() {
    int hv;
    lock {
        hv = h;
    }
    io:println(a, b, c, d, g, hv, i, j, k, l, m, n, xs, q, fp(1));
}
