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

// Two things about the file system that Ballerina cannot ask for on its own. These are not part of
// the ZIP surface in `natives.bal`: they are needed whatever implementation reads the archive, and
// they say nothing about ZIP files. What is done with them - what a resolved path is checked
// against, which time is written where - is decided in `extraction.bal`.

isolated function nativeRealPath(string path) returns string|Error = external;

isolated function nativeSetModifiedTime(string path, int seconds, int nanos) = external;

// Whether two paths are the same file. A resolved path answers this for links that carry a name of
// their own, but two hard links to one file are two names with nothing to resolve between them, so
// the file system has to be asked directly.
isolated function nativeIsSameFile(string first, string second) returns boolean|Error = external;
