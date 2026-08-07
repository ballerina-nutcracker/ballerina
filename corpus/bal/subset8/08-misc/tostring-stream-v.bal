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
    public isolated function next() returns record {|int value;|}|error? {
        int current = self.n;
        if current >= 3 {
            return ();
        }
        self.n = current + 1;
        return {value: current};
    }
}

public function main() {
    var s = new stream<int, error?>(new Counter());
    io:println(s); // @output stream
}
