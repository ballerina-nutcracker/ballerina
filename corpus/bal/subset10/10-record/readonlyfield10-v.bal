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

type Inner record {|
    int a?;
|};

type MR record {|
    readonly Inner value?;
    int otherValue;
|};

public function main() {
    MR mr = {value: {a: 7}, otherValue: 5};
    Inner? v = mr["value"];
    io:println(v); // @output {"a":7}
    io:println(mr["otherValue"]); // @output 5
    mr["otherValue"] = 9;
    io:println(mr.otherValue); // @output 9
}
