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

service class WatcherService {
    *file:Service;

    remote function onCreate(file:FileEvent m) {
        _ = m;
        lock {
            createCount = createCount + 1;
        }
    }
}

WatcherService watcherSvc = new;

function init() returns error? {
    check dirListener.attach(watcherSvc);
}

public function testMain() returns error? {
    string firstFile = check file:joinPath(watchDir, "one.txt");
    check file:create(firstFile);
    int countAfterFirst = 0;
    int attempts = 0;
    while attempts < 30 && countAfterFirst < 1 {
        lock {
            countAfterFirst = createCount;
        }
        if countAfterFirst < 1 {
            runtime:sleep(0.1);
        }
        attempts += 1;
    }
    io:println("countAfterFirst=", countAfterFirst); // @output countAfterFirst=1

    check dirListener.detach(watcherSvc);

    string secondFile = check file:joinPath(watchDir, "two.txt");
    check file:create(secondFile);
    // Detach is a negative assertion (no further dispatch), so there is no
    // "done" condition to poll for; observe a bounded window instead of a
    // single short sleep, giving a buggy still-attached watcher ample time
    // to have shown up as an unwanted extra increment.
    foreach int _ in 0 ..< 10 {
        runtime:sleep(0.1);
    }
    int countAfterDetach;
    lock {
        countAfterDetach = createCount;
    }
    io:println("countAfterDetach=", countAfterDetach); // @output countAfterDetach=1
}
