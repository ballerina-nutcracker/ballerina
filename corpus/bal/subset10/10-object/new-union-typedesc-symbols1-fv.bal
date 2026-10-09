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
class First {
    function init(any value, int marker) {
        var _ = value;
        var _ = marker;
    }
}

class Second {
    function init(any value, string marker) {
        var _ = value;
        var _ = marker;
    }
}

function double(int a) returns int => a * 2;

public function main() {
    any fn = double;
    map<int> m = {a: 2};
    First|Second a = new (fn is object { function run(); }, 1); // @error
    First|Second b = new (<function (int a) returns int>fn, 1); // @error
    First|Second c = new (<record {| int a = 1; |}>m, 1); // @error
    var _ = a;
    var _ = b;
    var _ = c;
}
