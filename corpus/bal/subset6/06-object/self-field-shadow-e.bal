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

class Foo {
    int bar = 5;

    function init(int x) {
        int _ = x;
        self.x = 10; // @error
    }

    function foo(int y) returns int {
        int _ = y;
        return self.y; // @error
    }

    function baz() returns int {
        int z = 10;
        int _ = z;
        return self.z; // @error
    }
}
