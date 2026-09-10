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

type Target record {|
    int x;
    string y;
|};

type Closed record {|
    int x;
|};

type OptionalX record {|
    int x?;
|};

type ExcludedX record {|
    never x?;
|};

public function main() {
    map<int> anyKey = {x: 1};
    Closed a = {...anyKey}; // @error inhabited source rest rejected by a closed target
    _ = a;

    OptionalX optional = {};
    Target c = {...optional, y: "hi"}; // @error an optional member does not establish presence
    _ = c;

    ExcludedX excluded = {};
    Target d = {...excluded, y: "hi"}; // @error an excluded member does not satisfy a required field
    _ = d;
}
