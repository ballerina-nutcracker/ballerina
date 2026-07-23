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

// noWriteParentDir: r-xr-xr-x, containing "existing.txt". Write-requiring
// operations on its entries (create/remove/rename a child) fail EACCES even
// though the entries themselves remain readable/statable.
isolated function noWriteParentDir() returns string = external;

// noExecParentDir: rw-------, containing "child.txt". Traversal into it
// (stat/readlink on a child) fails EACCES since execute is denied.
isolated function noExecParentDir() returns string = external;

// noAccessDir: no permission bits set at all. Listing its contents fails
// EACCES even though it can still be stat'd via its (accessible) parent.
isolated function noAccessDir() returns string = external;

public function testMain() returns error? {
    string noWriteParent = noWriteParentDir();
    string noExecParent = noExecParentDir();
    string noAccess = noAccessDir();

    // createDir fails when the parent directory denies write
    file:Error? createDirErr = file:createDir(noWriteParent + "/newsubdir");
    io:println(createDirErr is file:Error); // @output true

    // create fails when the parent directory denies write
    file:Error? createErr = file:create(noWriteParent + "/newfile.txt");
    io:println(createErr is file:Error); // @output true

    // rename fails when the source's parent directory denies write
    file:Error? renameErr = file:rename(noWriteParent + "/existing.txt", noWriteParent + "/renamed.txt");
    io:println(renameErr is file:Error); // @output true

    // remove fails when the parent directory denies write (entry is still stat-able)
    file:Error? removeErr = file:remove(noWriteParent + "/existing.txt");
    io:println(removeErr is file:Error); // @output true

    // remove fails when the parent directory denies traversal (entry can't even be stat'd)
    file:Error? removeNoExecErr = file:remove(noExecParent + "/child.txt");
    io:println(removeNoExecErr is file:Error); // @output true

    // normalizePath SYMLINK fails when the parent directory denies traversal
    string|file:Error resolveNoExecErr = file:normalizePath(noExecParent + "/child.txt", file:SYMLINK);
    io:println(resolveNoExecErr is file:Error); // @output true

    // readDir fails when the directory itself denies read/execute access
    file:MetaData[]|file:Error readDirErr = file:readDir(noAccess);
    io:println(readDirErr is file:Error); // @output true
}
