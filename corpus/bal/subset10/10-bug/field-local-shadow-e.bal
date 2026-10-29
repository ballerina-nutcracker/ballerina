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
class Buffer {
    int value = 0;

    function duplicate(int value) {
        var _ = value;
        int value = 1; // @error parameter is a block binding
        var _ = value;
    }

    function nested() {
        int value = 1;
        if value > 0 {
            int value = 2; // @error outer local is still a block binding
            var _ = value;
        }
    }

    function lambda() {
        int value = 1;
        function (int) returns int f = function (int value) returns int { // @error outer local blocks lambda parameter
            return value;
        };
        var _ = value + f(2);
    }

    function implicitSelf() {
        int self = 1; // @error implicit self is a block binding
        var _ = self;
    }
}

public function main() {
}
