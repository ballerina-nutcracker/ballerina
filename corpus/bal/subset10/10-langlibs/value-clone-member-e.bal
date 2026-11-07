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

class Counter {
    int n = 0;
}

type Holder record {|
    Counter c;
|};

public function main() {
    map<Counter> counters = {};
    _ = counters.clone(); // @error member type is not Cloneable
    Counter[] list = [];
    _ = list.clone(); // @error member type is not Cloneable
    Holder h = {c: new};
    _ = h.clone(); // @error field type is not Cloneable
    int|Counter u = 1;
    _ = u.clone(); // @error union with a non-Cloneable member
}
