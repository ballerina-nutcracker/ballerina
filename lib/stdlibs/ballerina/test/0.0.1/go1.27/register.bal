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

final TestRegistry testRegistry = new;
final TestRegistry beforeSuiteRegistry = new;
final TestRegistry afterSuiteRegistry = new;
final TestRegistry beforeEachRegistry = new;
final TestRegistry afterEachRegistry = new;

final GroupRegistry beforeGroupsRegistry = new;
final GroupRegistry afterGroupsRegistry = new;
final GroupStatusRegistry groupStatusRegistry = new;

// jballerina declares this in execute.bal (not yet ported, see TODO.md); moved
// here since the ExecutionManager class itself lives in this file.
final ExecutionManager executionManager = new;

// Workaround for a confirmed interpreter bug (see TODO.md): this compiler's
// unused-variable check is scoped per-file, not per-module — a real reference
// to one of the module-level vars above from a *different* file (filter.bal,
// execute.bal, annotation_processor.bal, all separate files per jballerina's
// own layout) never clears its "unused variable" error, even when that file's
// reference is itself reachable and used. This function is never called; its
// only job is to give each var a same-file reference so the (buggy) check
// passes. Remove once the compiler bug is fixed.
function retainModuleLevelRegistries() {
    _ = testRegistry;
    _ = beforeSuiteRegistry;
    _ = afterSuiteRegistry;
    _ = beforeEachRegistry;
    _ = afterEachRegistry;
    _ = beforeGroupsRegistry;
    _ = afterGroupsRegistry;
    _ = groupStatusRegistry;
    _ = executionManager;
}

// jballerina uses isolated/lock/readonly here for multi-strand safety; dropped
// since v1 never runs tests concurrently (see TODO.md).
type TestFunction record {|
    string name;
    function executableFunction;
    function? before = ();
    function? after = ();
    boolean alwaysRun = false;
    string[] groups = [];
    error? diagnostics = ();
    function[] dependsOn = [];
    boolean serialExecution = false;
    TestConfig? config = ();
|};

type TestFunctionMetaData record {|
    boolean enabled = true;
    boolean skip = false;
    int dependsOnCount = 0;
    TestFunction[] dependents = [];
    boolean visited = false;
    boolean isReadyToExecute = false;
    TestCompletionStatus executionCompletionStatus = YET_TO_COMPLETE;
|};

// Simplified from jballerina's dual parallel/serial queue + completion-polling
// design to a single FIFO ready-queue: execute.bal calls `onTestCompleted`
// synchronously after each serial test finishes, so there's nothing to poll.
// FIFO uses a plain array + read cursor since unshift/pop aren't implemented
// (see TODO.md).
class ExecutionManager {
    private final TestFunction[] readyQueue = [];
    private int readyQueueReadIndex = 0;
    private final map<TestFunctionMetaData> testMetaData = {};

    function createTestFunctionMetaData(string functionName, int dependsOnCount, boolean enabled) {
        self.testMetaData[functionName] = {dependsOnCount: dependsOnCount, enabled: enabled};
    }

    function setExecutionCompleted(string functionName) {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        if metaData is TestFunctionMetaData {
            metaData.executionCompletionStatus = COMPLETED;
        }
    }

    function setExecutionSuspended(string functionName) {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        if metaData is TestFunctionMetaData {
            metaData.executionCompletionStatus = SUSPENDED;
        }
    }

    function setDisabled(string functionName) {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        if metaData is TestFunctionMetaData {
            metaData.enabled = false;
        }
    }

    function setEnabled(string functionName) {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        if metaData is TestFunctionMetaData {
            metaData.enabled = true;
        }
    }

    function isEnabled(string functionName) returns boolean {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        return metaData is TestFunctionMetaData && metaData.enabled;
    }

    function setVisited(string functionName) {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        if metaData is TestFunctionMetaData {
            metaData.visited = true;
        }
    }

    function isVisited(string functionName) returns boolean {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        return metaData is TestFunctionMetaData && metaData.visited;
    }

    function isSkip(string functionName) returns boolean {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        return metaData is TestFunctionMetaData && metaData.skip;
    }

    function setSkip(string functionName) {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        if metaData is TestFunctionMetaData {
            metaData.skip = true;
        }
    }

    function addDependent(string functionName, TestFunction dependent) {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        if metaData is TestFunctionMetaData {
            metaData.dependents.push(dependent);
        }
    }

    function getDependents(string functionName) returns TestFunction[] {
        TestFunctionMetaData? metaData = self.testMetaData[functionName];
        if metaData is TestFunctionMetaData {
            return copyTestFunctionArray(metaData.dependents);
        }
        return [];
    }

    function addInitialTest(TestFunction testFunction) {
        self.readyQueue.push(testFunction);
    }

    function getQueueLength() returns int {
        return self.readyQueue.length() - self.readyQueueReadIndex;
    }

    function isExecutionDone() returns boolean {
        return self.readyQueueReadIndex >= self.readyQueue.length();
    }

    function getNextTest() returns TestFunction {
        TestFunction next = self.readyQueue[self.readyQueueReadIndex];
        self.readyQueueReadIndex += 1;
        return next;
    }

    // Replaces jballerina's populateExecutionQueues poll loop (no longer needed serially).
    function onTestCompleted(TestFunction testFunction) {
        TestFunctionMetaData? metaData = self.testMetaData[testFunction.name];
        if metaData is () {
            return;
        }
        metaData.executionCompletionStatus = COMPLETED;
        TestFunction[] dependents = metaData.dependents;
        int i = dependents.length() - 1;
        while i >= 0 {
            self.checkExecutionReadiness(dependents[i]);
            i -= 1;
        }
    }

    function onTestSuspended(TestFunction testFunction) {
        TestFunctionMetaData? metaData = self.testMetaData[testFunction.name];
        if metaData is TestFunctionMetaData {
            metaData.executionCompletionStatus = SUSPENDED;
        }
    }

    private function checkExecutionReadiness(TestFunction testFunction) {
        TestFunctionMetaData? metaData = self.testMetaData[testFunction.name];
        if metaData is () {
            return;
        }
        metaData.dependsOnCount -= 1;
        if metaData.dependsOnCount != 0 || metaData.isReadyToExecute {
            return;
        }
        metaData.isReadyToExecute = true;
        self.readyQueue.push(testFunction);
    }
}

class TestOptions {
    private string moduleName = "";
    private string packageName = "";
    private string targetPath = "";
    private map<string?> filterTestModules = {};
    private boolean hasFilteredTests = false;
    private string[] filterTests = [];
    private map<string[]> filterSubTests = {};

    function isFilterSubTestsContains(string key) returns boolean {
        return stringArrayMapHasKey(self.filterSubTests, key);
    }

    function getFilterSubTest(string key) returns string[] {
        string[]? subTests = self.filterSubTests[key];
        if subTests is string[] {
            return copyStringArray(subTests);
        }
        return [];
    }

    function addFilterSubTest(string key, string[] subTests) {
        self.filterSubTests[key] = copyStringArray(subTests);
    }

    function setFilterSubTests(map<string[]> filterSubTests) {
        self.filterSubTests = filterSubTests;
    }

    function setFilterTests(string[] filterTests) {
        self.filterTests = filterTests;
    }

    function addFilterTest(string filterTest) {
        self.filterTests.push(filterTest);
    }

    function getFilterTestSize() returns int {
        return self.filterTests.length();
    }

    function getFilterTestIndex(string testName) returns int? {
        return stringArrayIndexOf(self.filterTests, testName);
    }

    function getFilterTests() returns string[] {
        return copyStringArray(self.filterTests);
    }

    function setModuleName(string moduleName) {
        self.moduleName = moduleName;
    }

    function setPackageName(string packageName) {
        self.packageName = packageName;
    }

    function getModuleName() returns string {
        return self.moduleName;
    }

    function getPackageName() returns string {
        return self.packageName;
    }

    function setTargetPath(string targetPath) {
        self.targetPath = targetPath;
    }

    function getTargetPath() returns string {
        return self.targetPath;
    }

    function setFilterTestModules(map<string?> filterTestModulesMap) {
        self.filterTestModules = filterTestModulesMap;
    }

    function getFilterTestModule(string name) returns string? {
        return self.filterTestModules[name];
    }

    function setFilterTestModule(string key, string? value) {
        self.filterTestModules[key] = value;
    }

    function setHasFilteredTests(boolean hasFilteredTests) {
        self.hasFilteredTests = hasFilteredTests;
    }

    function getHasFilteredTests() returns boolean {
        return self.hasFilteredTests;
    }
}

class TestRegistry {
    private final TestFunction[] rootRegistry = [];
    private final TestFunction[] dependentRegistry = [];

    function addFunction(*TestFunction functionDetails) {
        if functionDetails.dependsOn == [] {
            self.rootRegistry.push(functionDetails);
        } else {
            self.dependentRegistry.push(functionDetails);
        }
    }

    function getTestFunction(function f) returns TestFunction|error {
        foreach TestFunction testFunction in self.rootRegistry {
            if f === testFunction.executableFunction {
                return testFunction;
            }
        }
        foreach TestFunction testFunction in self.dependentRegistry {
            if f === testFunction.executableFunction {
                return testFunction;
            }
        }
        return error("The dependent test function is either disabled or not included.");
    }

    function getFunctions() returns TestFunction[] {
        return sortTestFunctionsByName(self.rootRegistry);
    }

    function getDependentFunctions() returns TestFunction[] {
        return copyTestFunctionArray(self.dependentRegistry);
    }
}

class GroupRegistry {
    private final map<TestFunction[]> registry = {};

    function addFunction(string 'group, *TestFunction testFunction) {
        TestFunction[]? existing = self.registry['group];
        if existing is TestFunction[] {
            existing.push(testFunction);
        } else {
            self.registry['group] = [testFunction];
        }
    }

    function getFunctions(string 'group) returns TestFunction[]? {
        TestFunction[]? existing = self.registry['group];
        if existing is TestFunction[] {
            return copyTestFunctionArray(existing);
        }
        return;
    }
}

class GroupStatusRegistry {
    private final map<int> enabledTests = {};
    private final map<int> totalTests = {};
    private final map<int> executedTests = {};
    private final map<boolean> skip = {};

    function firstExecuted(string 'group) returns boolean {
        return intOrZero(self.executedTests['group]) > 0;
    }

    function lastExecuted(string 'group) returns boolean {
        return intOrZero(self.executedTests['group]) == intOrZero(self.enabledTests['group]);
    }

    function incrementTotalTest(string 'group, boolean enabled) {
        int? total = self.totalTests['group];
        if total is int {
            self.totalTests['group] = total + 1;
        } else {
            self.totalTests['group] = 1;
        }
        if enabled {
            self.skip['group] = false;
            int? enabledCount = self.enabledTests['group];
            if enabledCount is int {
                self.enabledTests['group] = enabledCount + 1;
            } else {
                self.enabledTests['group] = 1;
                self.executedTests['group] = 0;
            }
        }
    }

    function incrementExecutedTest(string 'group) {
        int? executed = self.executedTests['group];
        if executed is int {
            self.executedTests['group] = executed + 1;
        } else {
            self.executedTests['group] = 1;
        }
    }

    function setSkipAfterGroup(string 'group) {
        self.skip['group] = true;
    }

    function getSkipAfterGroup(string 'group) returns boolean {
        boolean? skipVal = self.skip['group];
        return skipVal is boolean && skipVal;
    }

    function getGroupsList() returns string[] {
        return copyStringArray(self.totalTests.keys());
    }
}

// Local stand-ins for missing lang.array/lang.map/lang.string methods (see
// TODO.md) — shared across filter.bal/execute.bal/report.bal as well as this
// file, hence living here rather than next to any one specific caller.

// Replaces `string:'join(sep, ...arr)` call sites — see TODO.md's rest-
// argument-spread entry (`...arr` at a call site is unimplemented).
function joinStrings(string sep, string[] arr) returns string {
    string result = "";
    foreach int i in 0 ..< arr.length() {
        if i > 0 {
            result += sep;
        }
        result += arr[i];
    }
    return result;
}

// Byte-level stand-ins for missing string:indexOf/includes/startsWith/endsWith
// (see TODO.md). Operate on bytes, not Unicode codepoints — fine for this
// module's own identifiers/messages (ASCII), not a general substitute.

function indexOfStr(string s, string target) returns int? {
    byte[] sBytes = s.toBytes();
    byte[] tBytes = target.toBytes();
    int tLen = tBytes.length();
    if tLen == 0 {
        return 0;
    }
    int sLen = sBytes.length();
    if tLen > sLen {
        return ();
    }
    foreach int i in 0 ... sLen - tLen {
        boolean matched = true;
        foreach int j in 0 ..< tLen {
            if sBytes[i + j] != tBytes[j] {
                matched = false;
                break;
            }
        }
        if matched {
            return i;
        }
    }
    return ();
}

function includesStr(string s, string target) returns boolean {
    return indexOfStr(s, target) is int;
}

function startsWithStr(string s, string prefix) returns boolean {
    byte[] sBytes = s.toBytes();
    byte[] pBytes = prefix.toBytes();
    if pBytes.length() > sBytes.length() {
        return false;
    }
    foreach int i in 0 ..< pBytes.length() {
        if sBytes[i] != pBytes[i] {
            return false;
        }
    }
    return true;
}

function endsWithStr(string s, string suffix) returns boolean {
    byte[] sBytes = s.toBytes();
    byte[] suffixBytes = suffix.toBytes();
    int offset = sBytes.length() - suffixBytes.length();
    if offset < 0 {
        return false;
    }
    foreach int i in 0 ..< suffixBytes.length() {
        if sBytes[offset + i] != suffixBytes[i] {
            return false;
        }
    }
    return true;
}

// Replaces `boolean:fromString(input)` (see TODO.md) — matches its actual
// semantics (only the exact literals "true"/"false" are accepted).
function parseBoolFromString(string input) returns boolean|error {
    if input == "true" {
        return true;
    }
    if input == "false" {
        return false;
    }
    return error("'" + input + "' is not a valid boolean value");
}

function intOrZero(int? n) returns int {
    if n is int {
        return n;
    }
    return 0;
}

function stringArrayMapHasKey(map<string[]> m, string k) returns boolean {
    foreach string existingKey in m.keys() {
        if existingKey == k {
            return true;
        }
    }
    return false;
}

function copyStringArray(string[] arr) returns string[] {
    string[] result = [];
    foreach string item in arr {
        result.push(item);
    }
    return result;
}

function copyTestFunctionArray(TestFunction[] arr) returns TestFunction[] {
    TestFunction[] result = [];
    foreach TestFunction item in arr {
        result.push(item);
    }
    return result;
}

function stringArrayIndexOf(string[] arr, string target) returns int? {
    foreach int i in 0 ..< arr.length() {
        if arr[i] == target {
            return i;
        }
    }
    return ();
}

// Replaces `arr.slice(startIndex)` (see TODO.md — array slicing isn't
// parseable/implemented).
function sliceFrom(string[] arr, int startIndex) returns string[] {
    string[] result = [];
    foreach int i in startIndex ..< arr.length() {
        result.push(arr[i]);
    }
    return result;
}

// Same as sliceFrom, for AnyOrError[][] — replaces `.remove(0)` (also missing)
// at serialExecuter.bal's data-driven-test queue-draining call sites.
function sliceFromAnyOrErrorArrays(AnyOrError[][] arr, int startIndex) returns AnyOrError[][] {
    AnyOrError[][] result = [];
    foreach int i in startIndex ..< arr.length() {
        result.push(arr[i]);
    }
    return result;
}

function sortTestFunctionsByName(TestFunction[] arr) returns TestFunction[] {
    TestFunction[] result = copyTestFunctionArray(arr);
    // Small insertion sort — test counts are small, no need for anything fancier.
    foreach int i in 1 ..< result.length() {
        TestFunction current = result[i];
        int j = i - 1;
        while j >= 0 && result[j].name > current.name {
            result[j + 1] = result[j];
            j -= 1;
        }
        result[j + 1] = current;
    }
    return result;
}
