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

function boom() {
    panic error("b panicked");
}

// trap catches the panic the flush re-raises, but the message is still
// unreceived, so the default worker terminates abnormally with b's panic.
function run() {
    worker b {
        boom();
        int _ = <- function;
    }
    1 -> b;
    error? e = trap flush b;
    io:println(e is error ? "trapped " + e.message() : "flushed"); // @output trapped b panicked
}

public function main() {
    error? r = trap run();
    io:println(r is error ? "run panicked: " + r.message() : "run done"); // @output run panicked: b panicked
}
