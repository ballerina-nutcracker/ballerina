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

listener file:Listener dirListener = checkpanic new ({path: watchDir, recursive: true});

isolated int createCount = 0;

service on dirListener {
    remote function onCreate(file:FileEvent m) {
        _ = m;
        lock {
            createCount = createCount + 1;
        }
    }
}

public function testMain() returns error? {
    string subDir = check file:joinPath(watchDir, "nested");
    check file:createDir(subDir);
    io:println("subDirCreateCount=", waitForCreateCount(1)); // @output subDirCreateCount=1

    string nestedFile = check file:joinPath(subDir, "inner.txt");
    check file:create(nestedFile);
    io:println("nestedFileCreateCount=", waitForCreateCount(2)); // @output nestedFileCreateCount=2
}

function waitForCreateCount(int expected) returns int {
    int count = 0;
    int attempts = 0;
    while attempts < 30 && count < expected {
        lock {
            count = createCount;
        }
        if count < expected {
            runtime:sleep(0.1);
        }
        attempts += 1;
    }
    return count;
}
