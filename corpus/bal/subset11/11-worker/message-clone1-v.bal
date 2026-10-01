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
    worker a returns int[] {
        int[] values = <- function;
        return values;
    }
    int[] values = [1, 2];
    values ->> a;
    values.push(3);
    int[] received = wait a;
    io:println(received); // @output [1,2]
    io:println(values); // @output [1,2,3]
}
