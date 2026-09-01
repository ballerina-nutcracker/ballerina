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
isolated string lastCreateName = "";

service on dirListener {
    remote function onCreate(file:FileEvent m) {
        lock {
            createInvoked = m.operation == "create";
        }
        lock {
            lastCreateName = m.name;
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
    // The lifecycle's $start already started dirListener; calling start()
    // again must be a harmless no-op (exercises the idempotent restart path).
    check dirListener.'start();

    check file:create(watchedFile);
    runtime:sleep(0.3);
    boolean createdSnapshot;
    lock {
        createdSnapshot = createInvoked;
    }
    string lastCreateNameSnapshot;
    lock {
        lastCreateNameSnapshot = lastCreateName;
    }
    io:println("created=", createdSnapshot); // @output created=true
    io:println("createdPathMatches=", lastCreateNameSnapshot == watchedFile); // @output createdPathMatches=true

    check file:copy("testdata/file-listener/fixture.txt", watchedFile, file:REPLACE_EXISTING);
    runtime:sleep(0.3);
    boolean modifiedSnapshot;
    lock {
        modifiedSnapshot = modifyInvoked;
    }
    io:println("modified=", modifiedSnapshot); // @output modified=true

    check file:remove(watchedFile);
    runtime:sleep(0.3);
    boolean deletedSnapshot;
    lock {
        deletedSnapshot = deleteInvoked;
    }
    io:println("deleted=", deletedSnapshot); // @output deleted=true
}
