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

isolated int[] guard = [1];

type Base object {
    int[] f;
};

class Derived {
    *Base;
    function init() {
        self.f = [];
    }
    function update() {
        lock {
            guard[0] = guard[0] + 1;
            self.f = guard; // @error target outside lock must be a plain variable name
        }
    }
}

public function main() {}
