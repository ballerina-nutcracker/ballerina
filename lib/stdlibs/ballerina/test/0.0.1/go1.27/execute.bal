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

boolean shouldSkip = false;
boolean shouldAfterSuiteSkip = false;
int exitCode = 0;
map<DataProviderReturnType?> dataDrivenTestParams = {};
decimal executionTime = 0;

// jballerina uses isolated/lock on shouldSkip/exitCode for multi-strand
// safety; dropped since v1 never runs tests concurrently (see TODO.md's
// isolated/lock entry, already applied the same way to register.bal/report.bal).

// Workaround for the confirmed per-file unused-variable check (see TODO.md):
// dataDrivenTestParams is read from serialExecuter.bal/registration_api.bal,
// but a cross-file reference never clears this check.
function retainExecuteModuleLevelVars() {
    _ = dataDrivenTestParams;
}

public function startSuite() returns int {
    if exitCode > 0 {
        return exitCode;
    }
    if listGroups {
        string[] groupsList = groupStatusRegistry.getGroupsList();
        if groupsList.length() == 0 {
            println("\tThere are no groups available!");
        } else {
            println("\t[" + joinStrings(", ", groupsList) + "]");
        }
    } else {
        if testRegistry.getFunctions().length() == 0 && testRegistry.getDependentFunctions().length() == 0 {
            println("\tNo tests found");
            return exitCode;
        }
        error? err = orderTests();
        if err is error {
            enableExit();
            println(err.message());
        } else {
            executeBeforeSuiteFunctions();
            err = executeTests();
            if err is error {
                enableExit();
                println(err.message());
            }
            executeAfterSuiteFunctions();
            foreach ReportGenerate reportGen in reportGenerators {
                reportGen(reportData);
            }
            println(string `${"\n"}${"\t"}${"\t"}Test execution time : ${executionTime / 1000}s`);
        }
    }
    return exitCode;
}

// Simplified from jballerina's dual parallel/serial-queue loop (see
// register.bal's ExecutionManager comment) to a single serial ready-queue —
// v1 ships serial execution only (see TODO.md's start/wait entry).
function executeTests() returns error? {
    decimal startTime = currentTimeInMillis();
    foreach TestFunction testFunction in testRegistry.getFunctions() {
        executionManager.addInitialTest(testFunction);
    }
    while !executionManager.isExecutionDone() {
        TestFunction testFunction = executionManager.getNextTest();
        executeTest(testFunction);
        executionManager.onTestCompleted(testFunction);
    }
    executionTime = currentTimeInMillis() - startTime;
}

function executeBeforeSuiteFunctions() {
    ExecutionError? err = executeFunctions(beforeSuiteRegistry.getFunctions());
    if err is ExecutionError {
        enableShouldSkip();
        shouldAfterSuiteSkip = true;
        enableExit();
        printExecutionError(err, "before test suite function");
    }
}

function executeAfterSuiteFunctions() {
    ExecutionError? err = executeFunctions(afterSuiteRegistry.getFunctions(), shouldAfterSuiteSkip);
    if err is ExecutionError {
        enableExit();
        printExecutionError(err, "after test suite function");
    }
}

// descendants is threaded as an immutable-per-call path (a fresh extended
// copy at each recursive step) instead of jballerina's shared push/pop stack
// — array push works but pop/slice don't exist yet (see TODO.md), and this
// achieves the identical cycle-detection semantics without needing either.
function orderTests() returns error? {
    foreach TestFunction testFunction in testRegistry.getDependentFunctions() {
        if !executionManager.isVisited(testFunction.name) && executionManager.isEnabled(testFunction.name) {
            check restructureTest(testFunction, []);
        }
    }
}

function restructureTest(TestFunction testFunction, string[] descendants) returns error? {
    string[] currentPath = [...descendants, testFunction.name];
    foreach function dependsOnFunction in testFunction.dependsOn {
        TestFunction dependsOnTestFunction = check testRegistry.getTestFunction(dependsOnFunction);

        // if the dependsOnFunction is disabled by the user, throw an error
        // dependsOnTestFunction.config?.enable is used instead of dependsOnTestFunction.enable to ensure that
        // the user has deliberately passed enable=false
        boolean? dependentEnabled = dependsOnTestFunction.config?.enable;
        if dependentEnabled == false {
            string errMsg = string `error: Test [${testFunction.name}] depends on function` +
            string ` [${dependsOnTestFunction.name}], but it is either disabled or not included.`;
            return error(errMsg);
        }
        executionManager.addDependent(dependsOnTestFunction.name, testFunction);

        // Contains cyclic dependencies
        int? startIndex = stringArrayIndexOf(currentPath, dependsOnTestFunction.name);
        if startIndex is int {
            string[] newCycle = sliceFrom(currentPath, startIndex);
            newCycle.push(dependsOnTestFunction.name);
            return error("Cyclic test dependencies detected: " + joinStrings(" -> ", newCycle));
        } else if !executionManager.isVisited(dependsOnTestFunction.name) {
            check restructureTest(dependsOnTestFunction, currentPath);
        }
    }
    executionManager.setEnabled(testFunction.name);
    executionManager.setVisited(testFunction.name);
}

function printExecutionError(ExecutionError err, string functionSuffix) {
    string[] parts = split(err.message(), "\n");
    string functionName = parts[0];
    string message = joinStrings("\n", sliceFrom(parts, 1));
    println("\t[fail] " + functionName + "[" + functionSuffix + "]" + ":\n\t    " +
            formatFailedError(message, 2));
}

// jballerina also appends a `\t`-indented stack trace (via `err.stackTrace()`,
// each frame's `toString()`) after the message. Neither `error:stackTrace()`
// nor a `StackFrame` type exist in this interpreter yet (confirmed via a
// standalone repro, not just langlib-file inspection) — see TODO.md. Messages
// are otherwise identical; only the trailing stack trace is missing.
function getErrorMessage(error err) returns string {
    return err.message() + "\n";
}

function getTestType(DataProviderReturnType? params) returns TestType {
    if params is map<AnyOrError[]> {
        return DATA_DRIVEN_MAP_OF_TUPLE;
    }
    if params is AnyOrError[][] {
        return DATA_DRIVEN_TUPLE_OF_TUPLE;
    }
    return GENERAL_TEST;
}

function nestedEnabledDependentsAvailable(TestFunction[] dependents) returns boolean {
    if dependents.length() == 0 {
        return false;
    }
    TestFunction[] queue = [];
    foreach TestFunction dependent in dependents {
        if executionManager.isEnabled(dependent.name) {
            return true;
        }
        foreach TestFunction superDependent in executionManager.getDependents(dependent.name) {
            queue.push(superDependent);
        }
    }
    return nestedEnabledDependentsAvailable(queue);
}

// Rewritten from an expression-bodied function (`=> expr;`) — the control-flow
// analyzer panics unconditionally on any BLangExprFunctionBody (confirmed via
// a standalone repro; see TODO.md). Applies to every `=>` function in this
// module, not just this one.
function isDataDrivenTest(DataProviderReturnType? params) returns boolean {
    return params is map<AnyOrError[]> || params is AnyOrError[][];
}

function enableShouldSkip() {
    shouldSkip = true;
}

function getShouldSkip() returns boolean {
    return shouldSkip;
}

function enableExit() {
    exitCode = 1;
}

function isTestReadyToExecute(TestFunction testFunction, DataProviderReturnType? testFunctionArgs) returns boolean {
    if !executionManager.isEnabled(testFunction.name) {
        executionManager.setExecutionSuspended(testFunction.name);
        return false;
    }
    error? diagnoseError = testFunction.diagnostics;
    if diagnoseError is error {
        reportData.onFailed(name = testFunction.name, message = diagnoseError.message(), testType =
                getTestType(testFunctionArgs));
        println(string `${"\n\t"}${testFunction.name} has failed.${"\n"}`);
        enableExit();
        executionManager.setExecutionSuspended(testFunction.name);
        return false;
    }
    return true;
}

function finishTestExecution(TestFunction testFunction, boolean shouldSkipDependents) {
    if shouldSkipDependents {
        foreach TestFunction dependent in executionManager.getDependents(testFunction.name) {
            executionManager.setSkip(dependent.name);
        }
    }
    executionManager.setExecutionCompleted(testFunction.name);
}

function handleBeforeGroupOutput(TestFunction testFunction, string 'group, ExecutionError? err) {
    if err is ExecutionError {
        executionManager.setSkip(testFunction.name);
        groupStatusRegistry.setSkipAfterGroup('group);
        enableExit();
        printExecutionError(err, "before groups function for the test");
    }
}

function handleBeforeEachOutput(ExecutionError? err) {
    if err is ExecutionError {
        enableShouldSkip();
        enableExit();
        printExecutionError(err, "before each function for the test");
    }
}

function handleNonDataDrivenTestOutput(TestFunction testFunction, ExecutionError|boolean output) returns boolean {
    boolean failed = false;
    if output is ExecutionError {
        failed = true;
        reportData.onFailed(name = testFunction.name, message = output.message(), testType = GENERAL_TEST);
        println(string `${"\n\t"}${testFunction.name} has failed.${"\n"}`);
    } else if output {
        failed = true;
    }
    return failed;
}

function handleAfterEachOutput(ExecutionError? err) {
    if err is ExecutionError {
        enableShouldSkip();
        enableExit();
        printExecutionError(err, "after each test function for the test");
    }
}

function handleAfterGroupOutput(ExecutionError? err) {
    if err is ExecutionError {
        enableExit();
        printExecutionError(err, "after test group function for the test");
    }
}

function handleDataDrivenTestOutput(ExecutionError|boolean err, TestFunction testFunction, string suffix,
        TestType testType) {
    if err is ExecutionError {
        reportData.onFailed(name = testFunction.name, suffix = suffix, message =
                string `[fail data provider for the function ${testFunction.name}]${"\n"} ${getErrorMessage(err)}`,
                testType = testType);
        println(string `${"\n\t"}${testFunction.name}:${suffix} has failed.${"\n"}`);
        enableExit();
    }
}

function handleBeforeFunctionOutput(ExecutionError? err) returns boolean {
    if err is ExecutionError {
        enableExit();
        printExecutionError(err, "before test function for the test");
        return true;
    }
    return false;
}

function handleAfterFunctionOutput(ExecutionError? err) returns boolean {
    if err is ExecutionError {
        enableExit();
        printExecutionError(err, "after test function for the test");
        return true;
    }
    return false;
}

function handleTestFuncOutput(any|error output, TestFunction testFunction, string suffix, TestType testType)
        returns ExecutionError|boolean {
    if output is TestError {
        enableExit();
        reportData.onFailed(name = testFunction.name, suffix = suffix, message = getErrorMessage(output),
                testType = testType);
        println(string `${"\n\t"}${testFunction.name}:${suffix} has failed.${"\n"}`);
        return true;
    }
    if output is any {
        reportData.onPassed(name = testFunction.name, suffix = suffix, testType = testType);
        return false;
    }
    enableExit();
    return error ExecutionError(testFunction.name + "\n" + getErrorMessage(output));
}

// Byte-level rewrite of jballerina's `foreach [string, AnyOrError[]] [k, v] in
// testFunctionArgs.entries()`/`foreach AnyOrError[] [k, v] in
// testFunctionArgs.enumerate()` — list-binding-pattern destructuring isn't
// implemented (see TODO.md, already worked around the same way in assert.bal).
function prepareDataSet(DataProviderReturnType? testFunctionArgs, string[] keys, AnyOrError[][] values)
        returns TestType {
    TestType testType = DATA_DRIVEN_MAP_OF_TUPLE;
    if testFunctionArgs is map<AnyOrError[]> {
        foreach string k in testFunctionArgs.keys() {
            keys.push(k);
            values.push(<AnyOrError[]>testFunctionArgs[k]);
        }
    } else if testFunctionArgs is AnyOrError[][] {
        testType = DATA_DRIVEN_TUPLE_OF_TUPLE;
        foreach int i in 0 ..< testFunctionArgs.length() {
            keys.push(string `${i}`);
            values.push(testFunctionArgs[i]);
        }
    }
    return testType;
}

function skipDataDrivenTest(TestFunction testFunction, string suffix, TestType testType) returns boolean {
    string functionName = testFunction.name;
    if !testOptions.getHasFilteredTests() {
        return false;
    }
    TestFunction[] dependents = executionManager.getDependents(functionName);

    // if a dependent in a below level is enabled, this test should run
    if dependents.length() > 0 && nestedEnabledDependentsAvailable(dependents) {
        return false;
    }
    string functionKey = functionName;

    // check if prefix matches directly
    boolean prefixMatch = testOptions.isFilterSubTestsContains(functionName);

    // if prefix matches to a wildcard
    if !prefixMatch && hasTest(functionName) {

        // get the matching wildcard
        prefixMatch = true;
        foreach string filter in testOptions.getFilterTests() {
            if includesStr(filter, WILDCARD) && matchWildcard(functionKey, filter) == true && matchModuleName(filter) {
                functionKey = filter;
                break;
            }
        }
    }

    // check if no filterSubTests found for a given prefix
    boolean suffixMatch = !testOptions.isFilterSubTestsContains(functionKey);

    // if a subtest is found specified
    if !suffixMatch {
        string[] subTests = testOptions.getFilterSubTest(functionKey);
        foreach string subFilter in subTests {
            string updatedSubFilter = subFilter;
            if testType == DATA_DRIVEN_MAP_OF_TUPLE && startsWithStr(subFilter, SINGLE_QUOTE)
                    && endsWithStr(subFilter, SINGLE_QUOTE) {
                updatedSubFilter = byteSubstring(subFilter, 1, subFilter.toBytes().length() - 1);
            }
            string|error decodedSubFilter = escapeSpecialCharacters(updatedSubFilter);
            if decodedSubFilter is string {
                updatedSubFilter = decodedSubFilter;
            }
            string|error decodedSuffix = escapeSpecialCharacters(suffix);
            string updatedSuffix = suffix;
            if decodedSuffix is string {
                updatedSuffix = decodedSuffix;
            }

            boolean wildCardMatch = false;
            if includesStr(updatedSubFilter, WILDCARD) {
                wildCardMatch = matchWildcard(updatedSuffix, updatedSubFilter) == true;
            }
            if (updatedSubFilter == updatedSuffix) || wildCardMatch {
                suffixMatch = true;
                break;
            }
        }
    }

    // do not skip iff both matches
    return !(prefixMatch && suffixMatch);
}

function isBeforeFuncConditionMet(TestFunction testFunction) returns boolean {
    return testFunction.before is function && !getShouldSkip() && !executionManager.isSkip(testFunction.name);
}

function isAfterFuncConditionMet(TestFunction testFunction) returns boolean {
    return testFunction.after is function && !getShouldSkip() && !executionManager.isSkip(testFunction.name);
}

function isSkipFunction(TestFunction testFunction) returns boolean {
    return executionManager.isSkip(testFunction.name) || getShouldSkip();
}
