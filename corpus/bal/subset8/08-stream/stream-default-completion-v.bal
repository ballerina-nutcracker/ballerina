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

    public isolated function next() returns record {|int value;|}? {
        int current = self.n;
        self.n = current + 1;
        return {value: current};
    }
}

type IntStream stream<int>;

type Holder record {|
    stream<int> s;
|};

public function main() {
    stream<int> a = new stream<int>(new Counter());
    io:println(a.next()); // @output {"value":0}

    IntStream b = new (new Counter());
    Holder h = {s: b};
    record {|int value;|}? r = h.s.next();
    io:println(r); // @output {"value":0}

    any x = a;
    io:println(x is stream<int>); // @output true
    io:println(x is stream<int, ()>); // @output true
    io:println(x is stream<int, error>); // @output false
}
