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

public function main() {
    int[] xs = [1, 2, 3, 4, 5];
    int total = 0;
    error? result = from var x in xs
        where x % 2 == 1
        let int y = x * 10
        limit 2
        do {
            total = total + y;
        };
    io:println(result is ()); // @output true
    io:println(total); // @output 40

    int firstTwo = 0;
    from var x in xs
        limit 2
        do {
            firstTwo = firstTwo + x;
        };
    io:println(firstTwo); // @output 3
}
