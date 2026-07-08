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

class Counter {
    int n = 0;

    public isolated function next() returns record {| int value; |}|() {
        int current = self.n;
        self.n = current + 1;
        if current >= 3 {
            return ();
        }
        return {value: current};
    }
}

class Doubler {
    Counter c = new Counter();

    public isolated function next() returns record {| int value; |}|() {
        record {| int value; |}|() r = self.c.next();
        if r is record {| int value; |} {
            return {value: r.value * 2};
        }
        return ();
    }
}

public function main() {
    stream<int, ()> doubled = new (new Doubler());
    record {| int value; |}|() r1 = doubled.next();
    if r1 is record {| int value; |} {
        io:println(r1.value); // @output 0
    }
    record {| int value; |}|() r2 = doubled.next();
    if r2 is record {| int value; |} {
        io:println(r2.value); // @output 2
    }
    record {| int value; |}|() r3 = doubled.next();
    if r3 is record {| int value; |} {
        io:println(r3.value); // @output 4
    }
    record {| int value; |}|() r4 = doubled.next();
    if r4 is () {
        io:println("done"); // @output done
    }
}
