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
    record {|
        boolean booleanField;
        int intField;
        float floatField;
    |} r = {
        intField: 17,
        floatField: 1.5,
        booleanField: false
    };
    io:println(r.floatField + 1.0); // @output 2.5
    io:println(!r.booleanField); // @output true
    io:println(r.intField + 2); // @output 19
}
