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

import ballerina/file;
import ballerina/io;
import ballerina/lang.runtime;

string watchDir = checkpanic file:createTempDir();

listener file:Listener dirListener = checkpanic new ({path: watchDir, recursive: false});

isolated boolean firstInvoked = false;
isolated boolean secondInvoked = false;

// The second service intentionally lacks onCreate: creating a file must
// skip it (no matching remote method) while still dispatching to the
// first service.
service on dirListener {
    remote function onCreate(file:FileEvent m) {
        _ = m;
        lock {
            firstInvoked = true;
        }
    }
}

service on dirListener {
    remote function onModify(file:FileEvent m) {
        _ = m;
        lock {
            secondInvoked = true;
        }
    }
}

public function testMain() returns error? {
    string filePath = check file:joinPath(watchDir, "sample.txt");
    check file:create(filePath);
    io:println("first=", waitFor(isFirstInvoked)); // @output first=true
    // Both services are dispatched for the same create event, so once the
    // first has run the second has already been skipped.
    io:println("secondAfterCreate=", isSecondInvoked()); // @output secondAfterCreate=false

    check io:fileWriteString(filePath, "modified");
    io:println("secondAfterModify=", waitFor(isSecondInvoked)); // @output secondAfterModify=true
}

function waitFor(function () returns boolean condition) returns boolean {
    int attempts = 0;
    while attempts < 30 {
        if condition() {
            return true;
        }
        runtime:sleep(0.1);
        attempts += 1;
    }
    return false;
}

function isFirstInvoked() returns boolean {
    lock {
        return firstInvoked;
    }
}

function isSecondInvoked() returns boolean {
    lock {
        return secondInvoked;
    }
}
