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

string[] filterGroups = [];
string[] filterDisableGroups = [];
boolean listGroups = false;
final TestOptions testOptions = new ();

// jballerina declares `hasTest`/`matchModuleName`/`hasGroup` in
// annotation_processor.bal (they're used by processConfigAnnotation's enable-
// gating and by execute.bal's skipDataDrivenTest). None of the three actually
// depend on `typeof` — they only read `testOptions`/`filterGroups`/
// `filterDisableGroups`, all of which live in this file — so they're moved
// here instead of living in the (not-ported, see TODO.md) annotation_processor.bal.
// This is a deliberate reorganization, not a behavior change.

function hasGroup(string[] groups, string[] filter) returns boolean {
    foreach string 'group in groups {
        if stringArrayIndexOf(filter, 'group) is int {
            return true;
        }
    }
    return false;
}

function hasTest(string name) returns boolean {
    if !testOptions.getHasFilteredTests() {
        return true;
    }
    int? testIndex = testOptions.getFilterTestIndex(name);
    if testIndex == () {
        foreach string filter in testOptions.getFilterTests() {
            if includesStr(filter, WILDCARD) && matchWildcard(name, filter) == true && matchModuleName(filter) {
                return true;
            }
        }
        return false;
    }
    return matchModuleName(name);
}

function matchModuleName(string testName) returns boolean {
    string? filterModule = testOptions.getFilterTestModule(testName);
    return filterModule == () || filterModule == getFullModuleName();
}

public function setTestOptions(string inTargetPath, string inPackageName, string inModuleName, string inReport,
        string inCoverage, string inGroups, string inDisableGroups, string inTests, string inRerunFailed,
        string inListGroups) {
    testOptions.setModuleName(inModuleName);
    testOptions.setPackageName(inPackageName);
    testOptions.setTargetPath(inTargetPath);
    filterGroups = parseStringArrayInput(inGroups);
    filterDisableGroups = parseStringArrayInput(inDisableGroups);
    boolean rerunFailed = parseBooleanInput(inRerunFailed, "rerun-failed");
    boolean testReport = parseBooleanInput(inReport, "test-report");
    boolean codeCoverage = parseBooleanInput(inCoverage, "code-coverage");
    listGroups = parseBooleanInput(inListGroups, "list-groups");

    if rerunFailed {
        error? err = parseRerunJson();
        if err is error {
            println("error: " + err.message());
            enableExit();
            return;
        }
        testOptions.setHasFilteredTests(true);
    } else {
        string[] singleExecTests = parseStringArrayInput(inTests);
        filterKeyBasedTests(inPackageName, inModuleName, singleExecTests);
        testOptions.setHasFilteredTests(testOptions.getFilterTestSize() > 0);
    }

    if testReport || codeCoverage {
        reportGenerators.push(moduleStatusReport);
    }
}

function parseStringArrayInput(string arrArg) returns string[] {
    if arrArg == "" {
        return [];
    }
    return split(arrArg, ",");
}

function filterKeyBasedTests(string packageName, string moduleName, string[] tests) {
    foreach string testName in tests {
        string updatedName = testName;
        string? prefix = ();
        if containsModulePrefix(packageName, moduleName, testName) {
            int separatorIndex = <int>indexOfStr(updatedName, MODULE_SEPARATOR);
            prefix = byteSubstring(updatedName, 0, separatorIndex);
            updatedName = byteSubstring(updatedName, separatorIndex + 1, updatedName.toBytes().length());
        }
        if containsDataKeySuffix(updatedName) {
            int separatorIndex = <int>indexOfStr(updatedName, DATA_KEY_SEPARATOR);
            string suffix = byteSubstring(updatedName, separatorIndex + 1, updatedName.toBytes().length());
            string testPart = byteSubstring(updatedName, 0, separatorIndex);
            if testOptions.isFilterSubTestsContains(testPart) {
                string[] subTestList = testOptions.getFilterSubTest(testPart);
                subTestList.push(suffix);
                testOptions.addFilterSubTest(testPart, subTestList);
            } else {
                testOptions.addFilterSubTest(testPart, [suffix]);
            }
            updatedName = testPart;
        }
        testOptions.addFilterTest(updatedName);
        testOptions.setFilterTestModule(updatedName, prefix);
    }
}

function parseBooleanInput(string input, string variableName) returns boolean {
    boolean|error booleanVariable = parseBoolFromString(input);
    if booleanVariable is error {
        println(string `Invalid '${variableName}' parameter: ${booleanVariable.message()}`);
        enableExit();
        return false;
    }
    return booleanVariable;
}

function parseRerunJson() returns error? {
    ModuleRerunJson|error moduleRerunJson = readModuleRerunEntry(
            testOptions.getTargetPath() + "/" + RERUN_JSON_FILE, testOptions.getModuleName());
    if moduleRerunJson is error {
        return error("error while running failed tests : " + moduleRerunJson.message());
    }
    testOptions.setFilterTests(moduleRerunJson.testNames);
    testOptions.setFilterTestModules(moduleRerunJson.testModuleNames);
    testOptions.setFilterSubTests(moduleRerunJson.subTestNames);
}

function containsModulePrefix(string packageName, string moduleName, string testName) returns boolean {
    return containsAPrefix(testName) && isPrefixInCorrectFormat(packageName, moduleName, testName);
}

function containsAPrefix(string testName) returns boolean {
    if includesStr(testName, MODULE_SEPARATOR) {
        if containsDataKeySuffix(testName) {
            return <int>indexOfStr(testName, MODULE_SEPARATOR) < <int>indexOfStr(testName, DATA_KEY_SEPARATOR);
        }
        return true;
    }
    return false;
}

function containsDataKeySuffix(string testName) returns boolean {
    return includesStr(testName, DATA_KEY_SEPARATOR);
}

function isPrefixInCorrectFormat(string packageName, string moduleName, string testName) returns boolean {
    string prefix = byteSubstring(testName, 0, <int>indexOfStr(testName, MODULE_SEPARATOR));
    return includesStr(prefix, packageName) || includesStr(prefix, packageName + DOT + moduleName);
}

// jballerina's own getFullModuleName concatenates a bare package name and a
// bare module-name-part (its TestOptions stores them separately). This
// port's cli/cmd/test.go passes the already-fully-qualified name (e.g.
// "pkg.submod", via projects.ModuleName.String()) as testOptions'
// moduleName for every module, default or named — see report.bal's own
// direct use of getModuleName() as a cache-directory/rerun-json key, which
// only works because it's already fully qualified. Re-concatenating the
// package name here would double it up for a named module (confirmed via a
// real repro: `--tests pkg.submod:testName` failed to match until querying
// with the doubled `pkg.pkg.submod:testName` instead) — so this just
// returns the already-correct value.
function getFullModuleName() returns string {
    return testOptions.getModuleName();
}
