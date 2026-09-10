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

type NoFields record {|
    never...;
|};

public function main() {
    map<int> ints = {};
    map<string> strings = {};
    var a = {...ints, ...strings}; // @error both rests are inhabited over the same key space
    _ = a;

    // An empty value does not shrink the static key space.
    map<int> left = {};
    map<int> right = {};
    var b = {...left, ...right}; // @error empty maps still share a key space
    _ = b;

    // A source rest can also land on a named field of another member.
    record {| int y; |} named = {y: 1};
    var c = {...ints, ...named}; // @error the map rest may supply 'y'
    _ = c;

    // An effective empty rest creates no overlap, so this pair is accepted.
    NoFields none = {};
    var d = {...none, ...ints};
    _ = d;
}
