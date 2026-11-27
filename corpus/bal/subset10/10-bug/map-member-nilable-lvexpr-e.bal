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

type R record {|
    int[]? xs?;
|};

type S record {|
    int a;
|};

function nilMap() returns map<int>? {
    return ();
}

function nilRecord() returns S? {
    return ();
}

public function main() {
    map<int[]?> ml = {};
    ml["k"][0] = 1; // @error
    map<map<int>?> mm = {};
    mm["k"]["j"] = 1; // @error
    map<int>? n = nilMap();
    n["x"] = 1; // @error
    R r = {};
    r.xs[0] = 1; // @error
    S? s = nilRecord();
    s.a = 1; // @error
}
