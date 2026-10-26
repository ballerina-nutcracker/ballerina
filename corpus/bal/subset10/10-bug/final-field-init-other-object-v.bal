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
import ballerina/io;

class Other {
    int x = 0;
}

class C {
    final int x;

    function init(Other o) {
        o.x = 1;
        self.x = 2;
        o.x = 3;
    }
}

public function main() {
    Other o = new;
    C c = new (o);
    io:println(c.x, " ", o.x); // @output 2 3
}
