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

class C {
    final int a = 1;
    int b = 0;
}

function write(C p) {
    p.b = 4;
    p.b += 1;
}

public function main() {
    C|int x = new C();
    if x is C {
        x.b = 2;
        write(x);
        io:println(x.b); // @output 5
        io:println(x.a); // @output 1
    }
}
