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

type NeverField record {|
    int x;
    never y?;
|};

type ReadonlyNeverField record {|
    int x;
    readonly never y?;
|};

public function main() {
    NeverField n = {x: 1};
    n.y = 2; // @error field can never be present, not a readonly violation
    n["y"] = 3; // @error field can never be present, not a readonly violation
    ReadonlyNeverField r = {x: 1};
    r.y = 4; // @error field can never be present, not a readonly violation
    r["y"] = 5; // @error field can never be present, not a readonly violation
}
