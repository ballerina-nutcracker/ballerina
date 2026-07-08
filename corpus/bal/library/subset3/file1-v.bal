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
    // getCurrentDir
    string cwd = file:getCurrentDir();
    io:println(cwd.length() > 0);

    // basename
    string base = check file:basename("/foo/bar/baz.txt");
    io:println(base);

    // parentPath
    string parent = check file:parentPath("/foo/bar/baz.txt");
    io:println(parent);

    // joinPath
    string joined = check file:joinPath("/foo", "bar", "baz");
    io:println(joined);

    // splitPath
    string[] parts = check file:splitPath("/foo/bar/baz");
    io:println(parts.length());
    io:println(parts[0]);
    io:println(parts[1]);
    io:println(parts[2]);

    // isAbsolutePath
    boolean absTrue = check file:isAbsolutePath("/foo/bar");
    io:println(absTrue);
    boolean absFalse = check file:isAbsolutePath("foo/bar");
    io:println(absFalse);

    // normalizePath CLEAN
    string cleaned = check file:normalizePath("/foo/../bar/./baz", file:CLEAN);
    io:println(cleaned);

    // relativePath
    string rel = check file:relativePath("/foo/bar", "/foo/bar/baz");
    io:println(rel);

    // getAbsolutePath
    string absPath = check file:getAbsolutePath(".");
    io:println(absPath.length() > 0);

    // createDir, test, remove
    check file:createDir("/tmp/bal-file-test-dir");
    boolean isDir = check file:test("/tmp/bal-file-test-dir", file:IS_DIR);
    io:println(isDir);
    check file:remove("/tmp/bal-file-test-dir");
    boolean gone = check file:test("/tmp/bal-file-test-dir", file:EXISTS);
    io:println(gone);

    // create, getMetaData, remove
    check file:create("/tmp/bal-file-test.txt");
    file:MetaData meta = check file:getMetaData("/tmp/bal-file-test.txt");
    io:println(meta.dir);
    io:println(meta.readable);
    check file:remove("/tmp/bal-file-test.txt");

    // createTemp / createTempDir
    string tmp = check file:createTemp(suffix = ".txt", prefix = "bal-");
    io:println(file:test(tmp, file:EXISTS));
    check file:remove(tmp);
    string tmpDir = check file:createTempDir(prefix = "bal-");
    io:println(file:test(tmpDir, file:IS_DIR));
    check file:remove(tmpDir);
}

// @output true
// @output baz.txt
// @output /foo/bar
// @output /foo/bar/baz
// @output 3
// @output foo
// @output bar
// @output baz
// @output true
// @output false
// @output /bar/baz
// @output baz
// @output true
// @output true
// @output false
// @output false
// @output true
// @output true
// @output true
