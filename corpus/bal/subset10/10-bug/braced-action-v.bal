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

client class Client {
    remote function value() returns int {
        return 7;
    }

    remote function broken() returns int {
        panic error("remote panic");
    }
}

function helper() returns int {
    return 5;
}

public function main() returns error? {
    Client c = new;

    future<int> first = start helper();
    int|error trapped = trap (wait first);
    io:println(trapped); // @output 5

    future<int> second = start helper();
    int|error waited = (wait second);
    io:println(waited); // @output 5

    var started = (start helper());
    int startedValue = check wait started;
    io:println(startedValue); // @output 5

    int remoteValue = (c->value());
    io:println(remoteValue); // @output 7

    future<int> third = start helper();
    int|error nested = trap (trap (wait third));
    io:println(nested); // @output 5

    int|error panicked = trap (c->broken());
    if panicked is error {
        io:println(panicked.message()); // @output remote panic
    }

    int checked = check (trap (c->value()));
    io:println(checked); // @output 7

    // Parentheses around an expression still group as before.
    io:println((1 + 2) * 3); // @output 9
}
