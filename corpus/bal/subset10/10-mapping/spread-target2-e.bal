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

type Closed record {|
    int x;
|};

type IntRestStringY record {|
    string y;
    int...;
|};

type OpenNeedsX record {|
    int x;
    int...;
|};

type Ambiguous record {|
    int x;
|};

type AlsoAmbiguous record {|
    int x;
|};

public function main() {
    // The extra named field is rejected by a closed target.
    record {| int x; int z; |} extra = {x: 1, z: 2};
    Closed a = {...extra}; // @error 'z' is not a field of the closed target
    _ = a;

    // The source rest fits the target rest but not the named target field.
    map<int> ints = {};
    IntRestStringY b = {...ints}; // @error the source rest may supply an int at 'y'
    _ = b;

    // Nothing guarantees the required field, although every possible key is admitted.
    record {| int z; |} onlyZ = {z: 1};
    OpenNeedsX f = {...onlyZ}; // @error 'x' is not guaranteed by the source
    _ = f;

    // A map source's rest is not admitted by the closed target.
    Closed c = {...ints}; // @error the map rest is not allowed by the closed target
    _ = c;

    Ambiguous|AlsoAmbiguous d = {x: 1}; // @error both alternatives apply
    _ = d;

    int|string e = {x: 1}; // @error no mapping type in the expected type
    _ = e;
}
