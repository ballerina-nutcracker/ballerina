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

// Exercises the native (palnative) FS.* closures NewPlatform wires up for
// ballerina/file (Getwd/Mkdir/MkdirAll/Remove/RemoveAll/Rename/CreateFile/
// Stat/Lstat/ReadDir/Copy/CreateTemp/CreateTempDir/Readlink) end-to-end
// through the real `bal` binary, so their coverage flows into the palnative
// profile. In-process corpus tests run under NewTestPal, which re-wires each
// of these closures itself (see test_harness.go) and never invokes
// NewPlatform's own copies, or the createParentDirs helper only reachable
// through them.
import ballerina/file;
import ballerina/io;

public function main() returns error? {
    string cwd = file:getCurrentDir();
    io:println(cwd.length() > 0); // @output true

    string baseDir = check file:createTempDir(prefix = "bal-cli-pal-file-");

    string dir = baseDir + "/dir";
    check file:createDir(dir);
    string nested = baseDir + "/nested/child";
    check file:createDir(nested, file:RECURSIVE);

    string f1 = dir + "/a.txt";
    check file:create(f1);
    file:MetaData meta = check file:getMetaData(f1);
    io:println(meta.dir); // @output false

    file:MetaData[] entries = check file:readDir(dir);
    io:println(entries.length()); // @output 1

    string f2 = dir + "/b.txt";
    check file:rename(f1, f2);
    io:println(check file:test(f2, file:EXISTS)); // @output true
    io:println(check file:test(f2, file:IS_SYMLINK)); // @output false

    string copyDest = baseDir + "/b-copy.txt";
    check file:copy(f2, copyDest);
    io:println(check file:test(copyDest, file:EXISTS)); // @output true

    // not a symlink, so this fails — still exercises the Readlink closure
    string|file:Error resolved = file:normalizePath(copyDest, file:SYMLINK);
    io:println(resolved is file:Error); // @output true

    string tempFile = check file:createTemp(prefix = "bal-cli-pal-file-", dir = baseDir);
    io:println(check file:test(tempFile, file:EXISTS)); // @output true

    check file:remove(baseDir, file:RECURSIVE);
    io:println(check file:test(baseDir, file:EXISTS)); // @output false
}
