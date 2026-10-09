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
class IntList {
    function init(int[] values, int marker) {
        var _ = values;
        var _ = marker;
    }
}

class StringList {
    function init(string[] values, string marker) {
        var _ = values;
        var _ = marker;
    }
}

public function main() {
    int[] values = [1, 2, 3];
    IntList|StringList l = new (from int v in values select v * 2, 1); // @error
    var _ = l;
}
