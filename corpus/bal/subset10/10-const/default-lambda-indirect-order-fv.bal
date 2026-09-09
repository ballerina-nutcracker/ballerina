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
    isolated function () returns S? f = isolated function () returns S? {
        return {r: {f: isolated function () returns S? => ()}, k: 5}; // @error future: S's default constructs R
    };
|};

type S record {|
    R r = {};
    int k = 1;
|};

public function main() {
    S s = {};
    var g = s.r.f;
    S? t = g();
    io:println(t is S ? t.k : -1);
}
