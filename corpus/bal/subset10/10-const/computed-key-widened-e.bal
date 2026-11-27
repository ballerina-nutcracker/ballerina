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
    int a;
|};

type S record {|
    int a;
    int flag?;
    string...;
|};

// The static key type admits a, but the folded key is outside the closed record.
const R BAD = {a: 1, [<string>"b"]: 2}; // @error
// The folded key selects a declared field, not the string rest type.
const S BAD_FIELD = {a: 1, [<string>"flag"]: "yes"}; // @error
// The folded key selects the string rest type, not one of the int fields.
const S BAD_REST = {a: 1, [<string>"extra"]: 2}; // @error

public function main() {
    _ = BAD;
    _ = BAD_FIELD;
    _ = BAD_REST;
}
