// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

public function main() returns error? {
    string baseDir = check file:createTempDir(prefix = "bal-file2-");
    string f1 = baseDir + "/a.txt";
    string f2 = baseDir + "/b.txt";
    check file:create(f1);
    check file:create(f2);

    // readDir
    file:MetaData[] entries = check file:readDir(baseDir);
    io:println(entries.length()); // @output 2

    // copy
    string copyDest = baseDir + "/a-copy.txt";
    check file:copy(f1, copyDest);
    boolean copyExists = check file:test(copyDest, file:EXISTS);
    io:println(copyExists); // @output true

    // copy onto an existing destination without REPLACE_EXISTING fails
    file:Error? copyNoReplace = file:copy(f2, copyDest);
    io:println(copyNoReplace is file:Error); // @output true

    // copy with REPLACE_EXISTING succeeds
    check file:copy(f2, copyDest, file:REPLACE_EXISTING);
    io:println(check file:test(copyDest, file:EXISTS)); // @output true

    // copy from a missing source fails
    file:Error? copyMissing = file:copy(baseDir + "/missing.txt", baseDir + "/x.txt");
    io:println(copyMissing is file:Error); // @output true

    // rename
    string renamed = baseDir + "/a-renamed.txt";
    check file:rename(f1, renamed);
    io:println(check file:test(renamed, file:EXISTS)); // @output true
    io:println(check file:test(f1, file:EXISTS)); // @output false

    // rename from a missing source fails
    file:Error? renameMissing = file:rename(f1, baseDir + "/y.txt");
    io:println(renameMissing is file:Error); // @output true

    // test READABLE / WRITABLE / IS_SYMLINK
    io:println(check file:test(f2, file:READABLE)); // @output true
    io:println(check file:test(f2, file:WRITABLE)); // @output true
    io:println(check file:test(f2, file:IS_SYMLINK)); // @output false

    // createDir on an existing directory fails
    file:Error? dirExistsErr = file:createDir(baseDir);
    io:println(dirExistsErr is file:Error); // @output true

    // recursive createDir succeeds for nested, non-existent parents
    string nested = baseDir + "/nested/child";
    check file:createDir(nested, file:RECURSIVE);
    io:println(check file:test(nested, file:IS_DIR)); // @output true

    // create on an existing file fails
    file:Error? createExistsErr = file:create(f2);
    io:println(createExistsErr is file:Error); // @output true

    // remove on a missing path fails
    file:Error? removeMissingErr = file:remove(baseDir + "/does-not-exist.txt");
    io:println(removeMissingErr is file:Error); // @output true

    // remove refuses to delete the current working directory
    string cwd = file:getCurrentDir();
    file:Error? removeCwdErr = file:remove(cwd);
    io:println(removeCwdErr is file:Error); // @output true

    // readDir on a non-directory path fails
    file:MetaData[]|file:Error readDirOnFile = file:readDir(f2);
    io:println(readDirOnFile is file:Error); // @output true

    // readDir on a missing path fails
    file:MetaData[]|file:Error readDirMissing = file:readDir(baseDir + "/does-not-exist");
    io:println(readDirMissing is file:Error); // @output true

    // getMetaData on a missing path fails
    file:MetaData|file:Error metaMissing = file:getMetaData(baseDir + "/does-not-exist.txt");
    io:println(metaMissing is file:Error); // @output true

    // createDir without RECURSIVE fails when the parent does not exist
    file:Error? dirNoParentErr = file:createDir(baseDir + "/no/such/parent");
    io:println(dirNoParentErr is file:Error); // @output true

    // createTemp/createTempDir fail when given a non-existent parent directory
    string|file:Error tempMissingDir = file:createTemp(dir = baseDir + "/does-not-exist");
    io:println(tempMissingDir is file:Error); // @output true
    string|file:Error tempDirMissingDir = file:createTempDir(dir = baseDir + "/does-not-exist");
    io:println(tempDirMissingDir is file:Error); // @output true

    // remove RECURSIVE cleans up everything
    check file:remove(baseDir, file:RECURSIVE);
    io:println(check file:test(baseDir, file:EXISTS)); // @output false
}
