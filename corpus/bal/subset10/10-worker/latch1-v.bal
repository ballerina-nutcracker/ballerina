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
import ballerina/lang.'__internal as internal;

// The startup latch is the compiler-internal primitive worker lowering builds
// on. An already-open latch returns without yielding, and opening a closed
// latch releases every waiter.
function waitForLatch(handle latch, string name) returns string {
    internal:waitOnLatch(latch);
    return name;
}

public function main() {
    handle opened = internal:createLatch();
    internal:openLatch(opened);
    internal:waitOnLatch(opened);
    io:println("already open returns immediately"); // @output already open returns immediately

    handle closed = internal:createLatch();
    future<string> first = start waitForLatch(closed, "first");
    future<string> second = start waitForLatch(closed, "second");
    io:println("waiters are still blocked"); // @output waiters are still blocked
    internal:openLatch(closed);
    string|error firstResult = wait first;
    string|error secondResult = wait second;
    io:println(firstResult); // @output first
    io:println(secondResult); // @output second
}
