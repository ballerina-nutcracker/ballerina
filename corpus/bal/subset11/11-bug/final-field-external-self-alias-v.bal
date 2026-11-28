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

    function reset() {
        C other = self;
        other.a = 5;
    }

    function init(boolean writeAlias = false) {
        if writeAlias {
            C other = self;
            other.a = 6;
        }
    }
}

public function main() {
    C|error initialized = trap new C(true);
    io:println(initialized is error); // @output true
    C c = new;
    error? reset = trap c.reset();
    io:println(reset is error); // @output true
    io:println(c.a); // @output 1
}
