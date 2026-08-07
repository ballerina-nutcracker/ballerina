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

type f function() returns int;

public function main() {
    int r = 0;
    int index = 0;
    f[] fs = [];
    while index < 10 {
        fs.push(function() returns int { return r; });
        r += index;
        index += 1;
    }

    int result = 0;
    foreach f func in fs {
        result += func();
    }
    io:println(result); // @output 450
}
