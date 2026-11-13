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
type Point record {|
    int x;
    int y;
|};

type OptionalX record {|
    int x?;
|};

public function main() {
    Point p = {x: 1, y: 2};
    var a = {...p, x: 3}; // @error spread before specific field
    _ = a;

    var b = {x: 3, ...p}; // @error specific field before spread
    _ = b;

    OptionalX o = {};
    var c = {x: 1, ...o}; // @error optional member can still collide
    _ = c;

    var d = {...p, ...p}; // @error two spreads share names
    _ = d;

    map<int> m = {};
    var e = {...m, ...m}; // @error two rest descriptors overlap
    _ = e;
}
