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

// Skipped on Windows and WASM (see test_util.WindowsUnsupportedTests,
// test_util.WASMUnsupportedTests): `echo` is a shell builtin on Windows and
// no executables are available in the WASM sandbox.
import ballerina/os;
import ballerina/io;

public function main() returns error? {
    // Run `echo`, wait for it to exit, and read its captured stdout.
    os:Process p = check os:exec({value: "echo", arguments: ["hello"]});
    int code = check p.waitForExit();
    io:println(code);             // @output 0
    byte[] out = check p.output();
    io:println(out.length() > 0); // @output true

    // The process's stderr stream is empty for a successful echo.
    os:Process p2 = check os:exec({value: "echo", arguments: ["world"]});
    int _ = check p2.waitForExit();
    byte[] err = check p2.output(io:stderr);
    io:println(err.length());     // @output 0

    // A directory whose name contains a space must reach `ls` as a single argv
    // entry (no shell word-splitting); listing it exits 0. Had the space been
    // split into "foo"/"bar", ls would receive two missing paths and fail.
    string dirWithSpace = "/tmp/bal_exec_space_test/foo bar";
    os:Process mk = check os:exec({value: "mkdir", arguments: ["-p", dirWithSpace]});
    io:println(check mk.waitForExit());  // @output 0
    os:Process lsProc = check os:exec({value: "ls", arguments: [dirWithSpace]});
    io:println(check lsProc.waitForExit());  // @output 0
    os:Process rm = check os:exec({value: "rm", arguments: ["-rf", "/tmp/bal_exec_space_test"]});
    int _ = check rm.waitForExit();

    // exec of a non-existent command returns an error.
    os:Process|os:Error bad = os:exec({value: "no_such_command_xyz_123"});
    io:println(bad is os:Error);  // @output true

    // exit() terminates a process.
    os:Process p3 = check os:exec({value: "echo", arguments: ["bye"]});
    p3.exit();
    io:println("exited");         // @output exited
}
