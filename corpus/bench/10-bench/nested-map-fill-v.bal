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

// @productions map-type-descriptor mapping-constructor-expr range-expr foreach-stmt assign-stmt local-var-decl-stmt member-access-expr int-literal
import ballerina/io;

public function main() {
    int total = 0;
    foreach int _ in 0 ..< 50 {
        map<map<int>> m = {};
        foreach int i in 0 ..< 20000 {
            m[i.toString()]["a"] = i;
        }
        total += m.length();
    }
    io:println(total); // @output 1000000
}
