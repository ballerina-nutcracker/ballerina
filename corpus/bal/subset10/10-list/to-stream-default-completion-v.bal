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

function makeStream(int[] values) returns stream<int> {
    return values.toStream();
}

function takeNext(stream<int> s) returns record {|int value;|}? {
    return s.next();
}

public function main() {
    stream<int> s = makeStream([1, 2]);
    io:println(takeNext(s)); // @output {"value":1}
    io:println(s.next()); // @output {"value":2}
    io:println(s.next() is ()); // @output true

    stream<int, ()> explicit = [3].toStream();
    stream<int> implicit = explicit;
    stream<int, ()> back = implicit;
    io:println(back.next()); // @output {"value":3}
}
