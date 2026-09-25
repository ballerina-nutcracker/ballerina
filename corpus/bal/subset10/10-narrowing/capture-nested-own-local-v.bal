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

// A lambda may narrow its own local before a nested lambda captures it.
public function main() {
    function () returns int outerFn = function () returns int {
        int|string x = 1;
        if x is int {
            int y = x;
            function () returns int nested = function () returns int {
                return x is int ? 1 : 2;
            };
            return y + nested();
        }
        return 0;
    };
    io:println(outerFn()); // @output 2
}
