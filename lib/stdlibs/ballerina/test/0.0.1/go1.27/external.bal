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

isolated function sprintf(string format, (any|error)... args) returns string = external;

isolated function getBallerinaType((any|error) value) returns string = external;

isolated function getStringDiff(string actual, string expected) returns string = external;

isolated function getKeysDiff(string[] actualKeys, string[] expectedKeys) returns string = external;

isolated function matchWildcard(string functionName, string functionPattern) returns boolean|error = external;

isolated function escapeSpecialCharacters(string key) returns string|error = external;

isolated function isSystemConsole() returns boolean = external;

isolated function currentTimeInMillis() returns decimal = external;

// jballerina's println (external.bal) writes each arg via a Java print(handle,
// obj) native under a lock. This port has no `handle`/System.out indirection —
// printValue writes straight to stdout — and no lock is needed since there's
// no shared mutable state to protect (see TODO.md's isolated/lock entry).
isolated function printValue(any|error obj) = external;

function println(anydata|error... objs) {
    foreach var obj in objs {
        printValue(obj);
    }
    printValue("\n");
}

// jballerina's split (external.bal) is a thin wrapper over Java's regex-based
// String#split. See native/test_io.go: this port's split is a literal
// (non-regex) split, which is behaviorally identical for this module's only
// call site (a literal "," delimiter in filter.bal).
public isolated function split(string receiver, string delimiter) returns string[] = external;

// Substitutes for jballerina's `function:call(fn, ...params)` — see TODO.md's
// `lang.function:call`/rest-argument-spread entries. Takes params as a plain
// array (not a rest parameter) so callers never need `...` spread.
isolated function invokeFunction(function func, AnyOrError[] params) returns any|error = external;

// Narrow, record-shaped substitutes for jballerina's generic `map<ModuleRerunJson>`
// + `json`/`.toString()`/`fromJsonStringWithType()` — see native/test_io.go's
// header comment and TODO.md's lang.value/JSON entry.
isolated function readModuleRerunEntry(string filePath, string moduleName) returns ModuleRerunJson|error = external;

isolated function writeModuleRerunEntry(string filePath, string moduleName, string[] testNames,
        map<string?> testModuleNames, map<string[]> subTestNames) returns error? = external;

isolated function writeModuleStatusReport(string filePath, int totalTests, int passed, int failed, int skipped,
        TestStatusEntry[] tests) returns error? = external;
