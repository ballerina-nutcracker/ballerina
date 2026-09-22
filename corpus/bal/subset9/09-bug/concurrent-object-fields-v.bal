// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/io;

isolated class Counters {
    private int counterA = 0;
    private int counterB = 0;

    isolated function incrementA() returns int {
        lock {
            while self.counterA < 10000 {
                self.counterA += 1;
            }
            return self.counterA;
        }
    }

    isolated function incrementB() returns int {
        lock {
            while self.counterB < 10000 {
                self.counterB += 1;
            }
            return self.counterB;
        }
    }
}

public function main() {
    Counters counters = new ();
    future<int> a = start counters.incrementA();
    future<int> b = start counters.incrementB();
    int|error aResult = wait a;
    int|error bResult = wait b;
    io:println(aResult); // @output 10000
    io:println(bResult); // @output 10000
}
