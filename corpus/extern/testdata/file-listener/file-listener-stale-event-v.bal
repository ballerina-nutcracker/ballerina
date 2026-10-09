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

isolated int createCount = 0;

// replayOnFirstWatch delivers a create event for path through the handler of
// the listener's first watch, standing in for an event that watch had already
// received when it was stopped.
isolated function replayOnFirstWatch(string path) = external;

service on dirListener {
    remote function onCreate(file:FileEvent m) {
        _ = m;
        lock {
            createCount += 1;
        }
    }
}

public function testMain() returns error? {
    string lateFile = check file:joinPath(watchDir, "late.txt");

    check dirListener.gracefulStop();
    replayOnFirstWatch(lateFile);
    io:println("afterStop=", currentCreateCount()); // @output afterStop=0

    check dirListener.'start();
    replayOnFirstWatch(lateFile);
    io:println("afterRestart=", currentCreateCount()); // @output afterRestart=0

    check file:create(lateFile);
    io:println("newWatch=", waitForCreateCount(1)); // @output newWatch=1
}

function currentCreateCount() returns int {
    lock {
        return createCount;
    }
}

function waitForCreateCount(int expected) returns int {
    int count = 0;
    int attempts = 0;
    while attempts < 30 && count < expected {
        count = currentCreateCount();
        if count < expected {
            runtime:sleep(0.1);
        }
        attempts += 1;
    }
    return count;
}
