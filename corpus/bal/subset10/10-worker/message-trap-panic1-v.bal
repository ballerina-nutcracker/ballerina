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

function boom() returns int {
    panic error("boom");
}

// A receive from a sender that panicked re-raises the panic, which trap
// catches; waiting on the sender afterwards still sees its panic.
public function main() {
    worker a {
        int x = boom();
        x -> function;
    }
    int|error r = trap <- a;
    io:println(r is error ? "trapped " + r.message() : "received"); // @output trapped boom
    error? w = trap wait a;
    io:println(w is error ? "trapped " + w.message() : "done"); // @output trapped boom
}
