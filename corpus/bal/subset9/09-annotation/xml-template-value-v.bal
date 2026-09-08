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

// A non-const annotation value need not be foldable: an xml template is a
// constant expression we cannot fold yet, so it must fall back to a runtime
// annotation global instead of being rejected.
type Info record {|
    xml value;
|};

annotation Info info on type;

const int I = 1;

@info {value: xml `<a>${I}</a>`}
type T int;

public function main() {
    Info? v = T.@info;
    if v is Info {
        io:println(v.value); // @output <a>1</a>
    }
}
