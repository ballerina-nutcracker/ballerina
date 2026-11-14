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
class P {
    int base = 3;

    function m(int a, int b = self.base) returns int => a + b; // @error

    function get(int a = self.base * 10) returns int { // @error
        return a;
    }

    function fieldDefault(int a = base) returns int => a; // @error

    function fieldBody() returns int => base; // @error

    function selfParam(int self) returns int => self; // @error
}

public function main() {
    P p = new;
    _ = p.m(1);
    _ = p.get();
    _ = p.fieldDefault();
    _ = p.fieldBody();
    _ = p.selfParam(1);
}
