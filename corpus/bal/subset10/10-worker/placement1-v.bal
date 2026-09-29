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
    int value;

    function init(int seed) {
        worker w returns int {
            return seed + 1;
        }
        self.value = wait w;
    }

    function doubled() returns int {
        worker w returns int {
            return self.value * 2;
        }
        return wait w;
    }
}

client class Greeter {
    resource function get greeting() returns string {
        worker w returns string {
            return "hello";
        }
        return wait w;
    }
}

public function main() {
    Counter counter = new (41);
    io:println(counter.value); // @output 42
    io:println(counter.doubled()); // @output 84

    Greeter greeter = new;
    string greeting = greeter->/greeting();
    io:println(greeting); // @output hello

    var anon = function() returns int {
        worker w returns int {
            return 5;
        }
        return wait w;
    };
    io:println(anon()); // @output 5
}
