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

// A loop in a lambda in P's default selects from P|Q while P's defaults are
// resolving: P fits through its defaults, so both members fit.
type P record {|
    int y = 0;
    int x = 1;
    isolated function () returns (P|Q)? f = isolated function () returns (P|Q)? {
        foreach int _ in 0 ..< 1 {
            return {y: 2}; // @error both P and Q are applicable
        }
        return ();
    };
|};

type Q record {|
    int y;
|};

public function main() {
    P p = {};
    var g = p.f;
    io:println(g());
}
