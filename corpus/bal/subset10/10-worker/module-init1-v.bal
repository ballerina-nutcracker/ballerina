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

isolated function spin() returns int {
    int i = 0;
    int acc = 0;
    while i < 5000 {
        acc += i;
        i += 1;
    }
    return acc;
}

final int moduleValue = spin();

// Module-level initialization runs before the module `init` function starts
// its named workers. The isolated `init` lets the worker run in parallel and
// read the variable while the declaring strand would still be in `spin`, so it
// would observe the variable unset if module initialization ran after startup.
isolated function init() {
    worker reader returns int {
        return moduleValue;
    }
    int observed = wait reader;
    io:println(observed == spin()); // @output true
}

public function main() {
}
