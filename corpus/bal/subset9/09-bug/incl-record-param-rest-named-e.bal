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

type StringConfig record {|
    string...;
|};

type IntConfig record {|
    int...;
|};

function withStringRest(*StringConfig config) {
    _ = config;
}

function withIntRest(*IntConfig config) {
    _ = config;
}

public function main() {
    withStringRest(name = "Alice", age = 30); // @error int is not a subtype of string rest parameter
    withIntRest(count = 1, label = "primary"); // @error string is not a subtype of int rest parameter
}
