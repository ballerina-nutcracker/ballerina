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

type meta record {|
    string label;
    int code;
|};

const string LABEL = "from-const";
const int CODE = 42;

annotation meta meta on type;

@meta {label: LABEL, code: CODE}
type Target record {|
    string id;
|};

public function main() {
    meta? value = Target.@meta;
    if value is meta {
        io:println(value.label); // @output from-const
        io:println(value.code); // @output 42
    }
}
