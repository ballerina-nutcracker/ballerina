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

class C { // @error
    function m() returns int {
        return 1;
    }
}

class C { // @error
    resource function get path() returns int { // @error
        return 1;
    }
}

client class K { // @error
}

client class K { // @error
    resource function get path() returns int {
        return 1;
    }
}

client class M { // @error
}

class M { // @error
    resource function get path() returns int { // @error
        return 1;
    }
}

class Q { // @error
}

client class Q { // @error
    resource function get path() returns int {
        return 1;
    }
}

class D { // @error
}

class D { // @error
    function m(typedesc<anydata> td = <>) returns td { // @error
        return 1;
    }
}

public function main() {
    C _ = new;
    K _ = new;
}
