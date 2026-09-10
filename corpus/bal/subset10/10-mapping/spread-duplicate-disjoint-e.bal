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
type IntX record {|
    int x;
|};

type StringX record {|
    string x;
|};

type ThirdY record {|
    int y;
|};

public function main() {
    IntX i = {x: 1};
    StringX s = {x: "a"};
    var a = {...i, ...s}; // @error names duplicate even with disjoint value types
    _ = a;

    ThirdY y = {y: 2};
    var b = {...i, ...y, ...s}; // @error nonadjacent spreads still collide
    _ = b;
}
