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

// Exercises the native (palnative) FS closures end-to-end through the real
// `bal` binary so WriteFile/AppendFile/ReadFile coverage flows into the
// palnative profile. In-process corpus tests run under NewTestPal, which wires
// os.ReadFile/WriteFile directly and never hits these closures.
//
// Unlike NewTestPal, the production PAL does not rewrite "/tmp" on Windows, so
// the target path is derived from the environment using the same precedence as
// Go's os.TempDir() (TMPDIR, then TMP/TEMP on Windows), falling back to /tmp.
import ballerina/io;
import ballerina/os;

public function main() returns error? {
    string dir = tempDir();
    string path = dir + "/bal_cli_pal_file.txt";
    check io:fileWriteString(path, "First");             // WriteFile (OVERWRITE)
    check io:fileWriteString(path, "Second", io:APPEND); // AppendFile
    io:println(check io:fileReadString(path));           // @output FirstSecond
}

function tempDir() returns string {
    foreach string name in ["TMPDIR", "TMP", "TEMP"] {
        string val = os:getEnv(name);
        if val != "" {
            return val;
        }
    }
    return "/tmp";
}
