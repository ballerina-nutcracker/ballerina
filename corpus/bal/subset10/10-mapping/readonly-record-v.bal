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

type R record {|
    int required;
    int defaulted = 5;
    int optional?;
    string...;
|};

public function main() {
    R withDefault = {readonly required: 1};
    io:println(withDefault); // @output {"required":1,"defaulted":5}

    R full = {readonly required: 1, defaulted: 2, readonly optional: 3, "rest": "r"};
    io:println(full); // @output {"required":1,"defaulted":2,"optional":3,"rest":"r"}
    full.defaulted = 20;
    io:println(full.defaulted); // @output 20
}
