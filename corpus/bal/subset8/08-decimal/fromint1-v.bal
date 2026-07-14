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
    io:println(<decimal>1); // @output 1
    io:println(<decimal>0); // @output 0
    io:println(<decimal>-1); // @output -1
    io:println(<decimal>2147483647); // @output 2147483647
    io:println(<decimal>2147483646); // @output 2147483646
    io:println(<decimal>2147483648); // @output 2147483648
    io:println(<decimal>-2147483648); // @output -2147483648
    io:println(<decimal>-2147483647); // @output -2147483647
    io:println(<decimal>-2147483649); // @output -2147483649
    io:println(<decimal>9223372036854775807); // @output 9223372036854775807
    io:println(<decimal>-9223372036854775808); // @output -9223372036854775808
}
