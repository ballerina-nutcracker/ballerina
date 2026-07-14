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

// @productions map-type-descriptor mapping-constructor-expr string-literal range-expr foreach-stmt type-cast-expr multiplicative-expr return-stmt additive-expr any function-call-expr assign-stmt local-var-decl-stmt int-literal
import ballerina/io;

public function main() {
    int max = 16;
    map<any> m = {};
    populate(m, max);
    io:println(retrieve(m, max)); // @output 1049008
}

function populate(map<any> m, int max) {
    string x = "x";
    int xLen = 1;

    foreach int _ in 0 ..< max {
        x = x + x;
        xLen = xLen * 2;
        m[x + ""] = xLen + 0;
        m[x + "a"] = xLen + 1;
        m[x + "ab"] = xLen + 2;
        m[x + "abc"] = xLen + 3;
        m[x + "abcd"] = xLen + 4;
        m[x + "abcde"] = xLen + 5;
        m[x + "abcdef"] = xLen + 6;
        m[x + "abcdefg"] = xLen + 7;
    }
}

function retrieve(map<any> m, int max) returns int {
    string x = "x";
    int res = 0;

    foreach int _ in 0 ..< max {
        x = x + x;
        res = res + (<int>m[x + ""])
                + (<int>m[x + "a"])
                + (<int>m[x + "ab"])
                + (<int>m[x + "abc"])
                + (<int>m[x + "abcd"])
                + (<int>m[x + "abcde"])
                + (<int>m[x + "abcdef"])
                + (<int>m[x + "abcdefg"]);
    }
    return res;
}
