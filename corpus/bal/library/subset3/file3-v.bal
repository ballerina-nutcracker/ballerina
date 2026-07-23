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
    string baseDir = check file:createTempDir(prefix = "bal-file3-");

    // rename onto an existing non-empty directory fails
    string emptyDir = baseDir + "/empty-dir";
    string nonEmptyDir = baseDir + "/non-empty-dir";
    check file:createDir(emptyDir);
    check file:createDir(nonEmptyDir);
    check file:create(nonEmptyDir + "/inside.txt");
    file:Error? renameOntoNonEmpty = file:rename(emptyDir, nonEmptyDir);
    io:println(renameOntoNonEmpty is file:Error); // @output true

    // rename into a non-existent parent directory fails
    string renameSrc = baseDir + "/rename-src.txt";
    check file:create(renameSrc);
    file:Error? renameNoParent = file:rename(renameSrc, baseDir + "/no/such/parent/dst.txt");
    io:println(renameNoParent is file:Error); // @output true

    // create fails when the parent directory doesn't exist
    file:Error? createNoParent = file:create(baseDir + "/no-such-parent/file.txt");
    io:println(createNoParent is file:Error); // @output true

    // create/getMetaData/readDir fail when a path component is a regular file, not a directory
    string regularFile = baseDir + "/regular.txt";
    check file:create(regularFile);
    file:Error? createThroughFile = file:create(regularFile + "/child.txt");
    io:println(createThroughFile is file:Error); // @output true
    file:MetaData|file:Error metaThroughFile = file:getMetaData(regularFile + "/child.txt");
    io:println(metaThroughFile is file:Error); // @output true
    file:MetaData[]|file:Error readDirThroughFile = file:readDir(regularFile + "/child.txt");
    io:println(readDirThroughFile is file:Error); // @output true

    // copy with COPY_ATTRIBUTES and NO_FOLLOW_LINKS options
    string copySrc = baseDir + "/copy-src.txt";
    check file:create(copySrc);
    string copyAttrDest = baseDir + "/copy-attr-dest.txt";
    check file:copy(copySrc, copyAttrDest, file:COPY_ATTRIBUTES);
    io:println(check file:test(copyAttrDest, file:EXISTS)); // @output true
    string copyNoFollowDest = baseDir + "/copy-nofollow-dest.txt";
    check file:copy(copySrc, copyNoFollowDest, file:NO_FOLLOW_LINKS);
    io:println(check file:test(copyNoFollowDest, file:EXISTS)); // @output true

    // copy to a destination whose parent directory doesn't exist fails
    file:Error? copyNoParent = file:copy(copySrc, baseDir + "/no-such-parent/dst.txt");
    io:println(copyNoParent is file:Error); // @output true

    // copy replicates a directory tree; re-copying with REPLACE_EXISTING
    // tolerates the destination (and its entries) already existing
    string copyDirSrc = baseDir + "/copy-dir-src";
    check file:createDir(copyDirSrc);
    check file:create(copyDirSrc + "/inside.txt");
    string copyDirDest = baseDir + "/copy-dir-dest";
    check file:copy(copyDirSrc, copyDirDest);
    io:println(check file:test(copyDirDest + "/inside.txt", file:EXISTS)); // @output true
    check file:copy(copyDirSrc, copyDirDest, file:REPLACE_EXISTING);
    io:println(check file:test(copyDirDest + "/inside.txt", file:EXISTS)); // @output true

    // createTempDir with an explicit suffix
    string tempDirWithSuffix = check file:createTempDir(suffix = "-suffix", prefix = "bal-", dir = baseDir);
    io:println(check file:test(tempDirWithSuffix, file:IS_DIR)); // @output true

    // test() on a non-existent path for every non-EXISTS option
    string missing = baseDir + "/does-not-exist";
    io:println(check file:test(missing, file:IS_DIR)); // @output false
    io:println(check file:test(missing, file:IS_SYMLINK)); // @output false
    io:println(check file:test(missing, file:READABLE)); // @output false
    io:println(check file:test(missing, file:WRITABLE)); // @output false

    // normalizePath SYMLINK on a non-existent path fails
    string|file:Error resolveMissing = file:normalizePath(missing, file:SYMLINK);
    io:println(resolveMissing is file:Error); // @output true

    // normalizePath SYMLINK on a regular (non-symlink) file fails
    string|file:Error resolveNotLink = file:normalizePath(regularFile, file:SYMLINK);
    io:println(resolveNotLink is file:Error); // @output true

    // remove without RECURSIVE fails on a non-empty directory
    string nonEmptyForRemove = baseDir + "/non-empty-for-remove";
    check file:createDir(nonEmptyForRemove);
    check file:create(nonEmptyForRemove + "/inside.txt");
    file:Error? removeNonEmptyErr = file:remove(nonEmptyForRemove);
    io:println(removeNonEmptyErr is file:Error); // @output true

    check file:remove(baseDir, file:RECURSIVE);
}
