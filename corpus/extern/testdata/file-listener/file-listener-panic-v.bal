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
string watchedFile = checkpanic file:joinPath(watchDir, "sample.txt");

listener file:Listener dirListener = checkpanic new ({path: watchDir, recursive: false});

isolated boolean createInvoked = false;
isolated boolean modifyInvoked = false;
isolated boolean deleteInvoked = false;
isolated int zero = 0;

service on dirListener {
    remote function onCreate(file:FileEvent m) {
        lock {
            createInvoked = m.operation == "create";
        }
        lock {
            int _ = 1 / zero;
        }
    }
    remote function onModify(file:FileEvent m) {
        lock {
            modifyInvoked = m.operation == "modify";
        }
    }
    remote function onDelete(file:FileEvent m) {
        lock {
            deleteInvoked = m.operation == "delete";
        }
    }
}

public function testMain() returns error? {
    check file:create(watchedFile);
    _ = waitFor(isCreateInvoked);

    check io:fileWriteString(watchedFile, "modified");
    io:println("modified=", waitFor(isModifyInvoked)); // @output modified=true

    check file:remove(watchedFile);
    io:println("deleted=", waitFor(isDeleteInvoked)); // @output deleted=true
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

function isCreateInvoked() returns boolean {
    lock {
        return createInvoked;
    }
}

function isModifyInvoked() returns boolean {
    lock {
        return modifyInvoked;
    }
}

function isDeleteInvoked() returns boolean {
    lock {
        return deleteInvoked;
    }
}
