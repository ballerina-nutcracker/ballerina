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

// symlinkPath: a real OS-level symlink named "link" pointing at the literal
// target "target-file.txt", created by the Go test fixture since the file
// module exposes no symlink-creation function.
isolated function symlinkPath() returns string = external;

// symlinkDirPath: a real directory containing a symlink named "link",
// pointing at the literal target "target-file.txt".
isolated function symlinkDirPath() returns string = external;

// dirLinkPath: the only entry of its parent directory, a symlink to a
// directory holding a single file "a.txt".
isolated function dirLinkPath() returns string = external;

// treeWithDirLinkPath: a real directory holding "sub-link", a symlink to the
// same directory dirLinkPath points at.
isolated function treeWithDirLinkPath() returns string = external;

// loopDirPath: a real directory holding "self", a symlink to itself.
isolated function loopDirPath() returns string = external;

public function testMain() returns error? {
    string link = symlinkPath();

    io:println(check file:test(link, file:IS_SYMLINK)); // @output true

    string target = check file:normalizePath(link, file:SYMLINK);
    io:println(target); // @output target-file.txt

    // copy with NO_FOLLOW_LINKS on a symlink source copies the link itself
    string linkCopyDest = check file:joinPath(check file:parentPath(link), "link-copy");
    check file:copy(link, linkCopyDest, file:NO_FOLLOW_LINKS);
    io:println(check file:test(linkCopyDest, file:IS_SYMLINK)); // @output true

    // copying a directory containing a symlink with NO_FOLLOW_LINKS
    // preserves the nested symlink instead of following it
    string symlinkDir = symlinkDirPath();
    string dirCopyDest = check file:joinPath(check file:parentPath(symlinkDir), "symlink-dir-copy");
    check file:copy(symlinkDir, dirCopyDest, file:NO_FOLLOW_LINKS);
    string copiedLink = check file:joinPath(dirCopyDest, "link");
    io:println(check file:test(copiedLink, file:IS_SYMLINK)); // @output true

    // readDir reports a symlinked directory using its target's metadata
    string dirLink = dirLinkPath();
    file:MetaData[] entries = check file:readDir(check file:parentPath(dirLink));
    io:println(entries.length(), " ", entries[0].dir); // @output 1 true

    // following a symlinked directory copies the target's tree; jBallerina
    // produces an empty directory here, a bug the README records
    string dirLinkCopy = check file:joinPath(check file:parentPath(dirLink), "dir-link-copy");
    check file:copy(dirLink, dirLinkCopy);
    io:println(check file:test(dirLinkCopy, file:IS_SYMLINK)); // @output false
    io:println(check file:test(check file:joinPath(dirLinkCopy, "a.txt"), file:EXISTS)); // @output true

    // a symlinked subdirectory inside a copied tree is followed the same way
    string tree = treeWithDirLinkPath();
    string treeCopy = check file:joinPath(check file:parentPath(tree), "tree-copy");
    check file:copy(tree, treeCopy);
    string copiedSub = check file:joinPath(treeCopy, "sub-link");
    io:println(check file:test(copiedSub, file:IS_SYMLINK)); // @output false
    io:println(check file:test(check file:joinPath(copiedSub, "a.txt"), file:EXISTS)); // @output true

    // following a symlink that loops back to an ancestor fails instead of
    // recursing forever
    string loop = loopDirPath();
    file:Error? loopErr = file:copy(loop, check file:joinPath(check file:parentPath(loop), "loop-copy"));
    io:println(loopErr is file:Error); // @output true
}
