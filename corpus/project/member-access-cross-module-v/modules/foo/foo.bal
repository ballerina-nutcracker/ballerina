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

public type T object {
    public int pub;
    public function get() returns int;
};

public class C {
    private int secret = 3;
    int mod = 4;
    public int y = 2;

    private function hidden() returns int {
        return self.secret;
    }

    public function get() returns int {
        return self.hidden() + self.mod;
    }

    public function peek(C other) returns int {
        return other.secret + other.hidden();
    }
}

public client class Cl {
    remote function r() returns int {
        return 9;
    }
}

public function readMod(C c) returns int {
    return c.mod;
}
