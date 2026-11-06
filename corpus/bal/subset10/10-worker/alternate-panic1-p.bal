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

// The other worker can only produce an error, so the alternate wait cannot
// complete before the panicking worker terminates, and it completes abruptly.
public function main() {
    worker failing returns int|error {
        panic error("boom"); // @panic
    }
    worker failed returns int|error {
        return error("normal failure");
    }
    int|error r = wait failing | failed;
    io:println(r);
}
