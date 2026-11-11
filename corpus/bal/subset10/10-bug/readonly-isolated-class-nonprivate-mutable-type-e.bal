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

// Field types of a readonly class are not yet intersected with readonly (#1043),
// so a non-private field of a mutable type is rejected.
readonly isolated class P {
    int[] xs = [1, 2]; // @error mutable list type in a readonly isolated class
    public map<int> m = {}; // @error mutable map type in a readonly isolated class
    private int[] hidden = [];
    string name = "p";
}

public function main() {
    P _ = new;
}
