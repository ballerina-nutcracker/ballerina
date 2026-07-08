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

class Emit {
    int n = 0;

    public isolated function next() returns record {| int value; |}|() {
        if self.n >= 2 {
            return ();
        }
        int current = self.n;
        self.n = current + 1;
        return {value: current};
    }
}

public function main() {
    stream<int, ()> s = new (new Emit());
    () c = s.close();
    io:println(c); // @output
    record {| int value; |}|() r = s.next();
    if r is record {| int value; |} {
        io:println(r.value); // @output 0
    }
}
