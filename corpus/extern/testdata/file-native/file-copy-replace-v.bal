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

// sourcePath: a regular file holding "hello\n".
isolated function sourcePath() returns string = external;

// hardLinkPath: a hard link to the source file.
isolated function hardLinkPath() returns string = external;

// symlinkPath: a symlink whose target is the source file.
isolated function symlinkPath() returns string = external;

// replaceDstPath: a regular file holding "old\n", sharing its inode with
// replaceDstHardLinkPath.
isolated function replaceDstPath() returns string = external;

isolated function replaceDstHardLinkPath() returns string = external;

public function testMain() returns error? {
    string src = sourcePath();

    // copying a file onto itself is a no-op, with or without REPLACE_EXISTING
    check file:copy(src, src);
    check file:copy(src, src, file:REPLACE_EXISTING);
    io:println((check file:getMetaData(src)).size); // @output 6

    // a hard link to the source is the same file
    check file:copy(src, hardLinkPath());
    io:println((check file:getMetaData(src)).size); // @output 6

    // replacing a symlink to the source replaces the link, not the source
    string link = symlinkPath();
    check file:copy(src, link, file:REPLACE_EXISTING);
    io:println((check file:getMetaData(src)).size); // @output 6
    io:println(check file:test(link, file:IS_SYMLINK)); // @output false

    // like Java's Files.copy, REPLACE_EXISTING deletes the destination before
    // copying, so other hard links to it keep the old content
    string dst = replaceDstPath();
    check file:copy(src, dst, file:REPLACE_EXISTING);
    io:println((check file:getMetaData(dst)).size, " ", (check file:getMetaData(replaceDstHardLinkPath())).size); // @output 6 4

    // an empty directory at the destination is replaced; a non-empty one is not
    string emptyDir = check file:joinPath(check file:parentPath(src), "empty-dir");
    check file:createDir(emptyDir);
    check file:copy(src, emptyDir, file:REPLACE_EXISTING);
    io:println(check file:test(emptyDir, file:IS_DIR)); // @output false
    string fullDir = check file:joinPath(check file:parentPath(src), "full-dir");
    check file:createDir(fullDir);
    check file:create(check file:joinPath(fullDir, "inside.txt"));
    file:Error? fullDirErr = file:copy(src, fullDir, file:REPLACE_EXISTING);
    io:println(fullDirErr is file:Error); // @output true
}
