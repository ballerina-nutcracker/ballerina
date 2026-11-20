// Copyright (c) 2026 WSO2 LLC. (http://www.wso2.com).
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

// What both sides of the library ask of the file system, with the errors of the `file` module turned
// into the errors of this one. Nothing here knows about ZIP files.

isolated function realPath(string path) returns string|Error {
    return nativeRealPath(path);
}

isolated function isSameFile(string first, string second) returns boolean|Error {
    return nativeIsSameFile(first, second);
}

// The prefix a path must carry to sit under `root`. A root that is already a separator, which is what
// a file system root resolves to, would otherwise be doubled and match nothing under it.
isolated function withinPrefix(string root) returns string {
    return hasSuffix(root, file:pathSeparator) ? root : root + file:pathSeparator;
}

isolated function pathExists(string path) returns boolean {
    boolean|file:Error exists = file:test(path, file:EXISTS);
    if exists is boolean && exists {
        return true;
    }
    // `file:EXISTS` follows links, so a link whose target is missing would be reported as nothing
    // being there and would then be written through, placing the content wherever it points. A link
    // occupying a path counts as the path being occupied.
    return isSymlink(path);
}

isolated function isDirectory(string path) returns boolean {
    boolean|file:Error directory = file:test(path, file:IS_DIR);
    return directory is boolean && directory;
}

isolated function isSymlink(string path) returns boolean {
    boolean|file:Error link = file:test(path, file:IS_SYMLINK);
    return link is boolean && link;
}

isolated function parentOf(string path) returns string|Error {
    string|file:Error parent = file:parentPath(path);
    if parent is file:Error {
        return error FileSystemError(string `cannot resolve the parent directory of '${path}'`);
    }
    return parent;
}

isolated function baseNameOf(string path) returns string|Error {
    string|file:Error name = file:basename(path);
    if name is file:Error {
        return error FileSystemError(string `cannot resolve the file name of '${path}'`);
    }
    return name;
}

// Takes two parts, where jBallerina takes a rest parameter, since the interpreter cannot yet spread a
// list into a call (ballerina-nutcracker/ballerina#860).
isolated function joinPath(string first, string second) returns string|Error {
    string|file:Error joined = file:joinPath(first, second);
    if joined is file:Error {
        return error FileSystemError(string `cannot resolve the path '${first}/${second}'`);
    }
    return joined;
}
