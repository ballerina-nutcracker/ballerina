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

const string RERUN_JSON_FILE = "rerun_test.json";
const string MODULE_STATUS_JSON_FILE = "module_status.json";
const string CACHE_DIRECTORY = "cache";
const string TESTS_CACHE_DIRECTORY = "tests_cache";

type ReportGenerate function (ReportData data);

final ReportData reportData = new ();

ReportGenerate[] reportGenerators = [consoleReport, failedTestsReport];

// Workaround for the confirmed per-file (not per-module) unused-variable
// check (see TODO.md, same pattern as register.bal's retainModuleLevelRegistries):
// reportData/reportGenerators are read from execute.bal/filter.bal, but a
// cross-file reference never clears this check — only a same-file one does.
function retainReportModuleLevelVars() {
    _ = reportData;
    _ = reportGenerators;
}

// jballerina uses isolated/lock/readonly here for multi-strand safety; dropped
// since v1 never runs tests concurrently (see TODO.md's isolated/lock entry —
// the same decision already applied to register.bal, extended here).
type ResultData record {|
    string name;
    string suffix = "";
    string message = "";
    TestType testType;
|};

class Result {
    private ResultData data;

    function init(ResultData data) {
        self.data = data;
    }

    function fullName() returns string {
        if self.data.suffix == "" {
            return self.data.name;
        }
        return self.data.name + DATA_KEY_SEPARATOR + self.data.suffix;
    }

    function isDataProvider() returns boolean {
        return self.data.suffix != "";
    }

    function testPrefix() returns string {
        return self.data.name;
    }

    function testSuffix() returns string {
        return self.data.suffix;
    }

    function message() returns string {
        return self.data.message;
    }

    function testType() returns TestType {
        return self.data.testType;
    }
}

class ReportData {
    private final ResultData[] passed = [];
    private final ResultData[] failed = [];
    private final ResultData[] skipped = [];

    function onPassed(*ResultData result) {
        self.passed.push(result);
    }

    function onFailed(*ResultData result) {
        self.failed.push(result);
    }

    function onSkipped(*ResultData result) {
        self.skipped.push(result);
    }

    function passedCases() returns ResultData[] {
        return copyResultDataArray(self.passed);
    }

    function failedCases() returns ResultData[] {
        return copyResultDataArray(self.failed);
    }

    function skippedCases() returns ResultData[] {
        return copyResultDataArray(self.skipped);
    }

    function passedCount() returns int {
        return self.passed.length();
    }

    function failedCount() returns int {
        return self.failed.length();
    }

    function skippedCount() returns int {
        return self.skipped.length();
    }
}

function copyResultDataArray(ResultData[] arr) returns ResultData[] {
    ResultData[] result = [];
    foreach ResultData item in arr {
        result.push(item);
    }
    return result;
}

function consoleReport(ReportData data) {
    if !isSystemConsole() {
        foreach ResultData entrydata in data.passedCases() {
            Result entry = new (entrydata);
            println("\t\t[pass] " + entry.fullName());
        }
    }

    foreach ResultData entrydata in data.failedCases() {
        Result entry = new (entrydata);
        println("\n\t\t[fail] " + entry.fullName() + ":");
        println("\n\t\t    " + formatFailedError(entry.message(), 3));
    }

    int totalTestCount = data.passedCount() + data.failedCount() + data.skippedCount();

    println("\n");
    if totalTestCount == 0 {
        println("\t\tNo tests found");
    } else {
        println(string `${"\t"}${"\t"}${data.passedCount()} passing`);
        println(string `${"\t"}${"\t"}${data.failedCount()} failing`);
        println(string `${"\t"}${"\t"}${data.skippedCount()} skipped`);
    }
}

function formatFailedError(string message, int tabCount) returns string {
    string[] lines = split(message, "\n");
    lines.push("");
    string tabs = "";
    foreach int _ in 1 ... tabCount {
        tabs += "\t";
    }
    return joinStrings("\n" + tabs, lines);
}

function failedTestsReport(ReportData data) {
    string[] testNames = [];
    map<string?> testModuleNames = {};
    map<string[]> subTestNames = {};
    foreach ResultData resultdata in data.failedCases() {
        Result result = new (resultdata);
        string testPrefix = result.testPrefix();
        string testSuffix;
        if result.testType() == DATA_DRIVEN_MAP_OF_TUPLE {
            testSuffix = SINGLE_QUOTE + result.testSuffix() + SINGLE_QUOTE;
        } else {
            testSuffix = result.testSuffix();
        }
        testNames.push(testPrefix);
        testModuleNames[testPrefix] = testOptions.getModuleName();
        if result.isDataProvider() {
            string[]? existing = subTestNames[testPrefix];
            if existing is string[] {
                existing.push(testSuffix);
                subTestNames[testPrefix] = existing;
            } else {
                subTestNames[testPrefix] = [testSuffix];
            }
        }
    }
    string filePath = testOptions.getTargetPath() + "/" + RERUN_JSON_FILE;
    error? err = writeModuleRerunEntry(filePath, testOptions.getModuleName(), testNames, testModuleNames,
            subTestNames);
    if err is error {
        println(err.message());
    }
}

function moduleStatusReport(ReportData data) {
    TestStatusEntry[] tests = [];
    foreach ResultData resultdata in data.passedCases() {
        tests.push({name: escapeSpecialCharactersJson(new Result(resultdata).fullName()), status: "PASSED"});
    }
    foreach ResultData resultdata in data.failedCases() {
        Result result = new (resultdata);
        tests.push({
            name: escapeSpecialCharactersJson(result.fullName()),
            status: "FAILURE",
            failureMessage: replaceDoubleQuotes(result.message())
        });
    }
    foreach ResultData resultdata in data.skippedCases() {
        tests.push({name: escapeSpecialCharactersJson(new Result(resultdata).fullName()), status: "SKIPPED"});
    }

    int totalTests = data.passedCount() + data.failedCount() + data.skippedCount();
    string filePath = testOptions.getTargetPath() + "/" + CACHE_DIRECTORY + "/" + TESTS_CACHE_DIRECTORY
                + "/" + testOptions.getModuleName() + "/" + MODULE_STATUS_JSON_FILE;
    error? err = writeModuleStatusReport(filePath, totalTests, data.passedCount(), data.failedCount(),
            data.skippedCount(), tests);
    if err is error {
        println(err.message());
    }
}

function escapeSpecialCharactersJson(string name) returns string {
    string|error encodedName = escapeSpecialCharacters(name);
    if encodedName is string {
        return encodedName;
    }
    return name;
}

// Byte-level rewrite of jballerina's `foreach string chr in originalString` —
// strings aren't Iterable in this interpreter yet (see TODO.md). Escaping
// itself stays byte-level (the quote character is single-byte ASCII either
// way), but the bytes are accumulated whole and decoded once at the end,
// since decoding one byte at a time breaks on any multi-byte UTF-8 character
// — a lone byte from the middle of one isn't valid UTF-8 by itself.
function replaceDoubleQuotes(string originalString) returns string {
    byte[] bytes = originalString.toBytes();
    byte[] updatedBytes = [];
    foreach byte b in bytes {
        if b == 34 {
            updatedBytes.push(92);
            updatedBytes.push(34);
        } else {
            updatedBytes.push(b);
        }
    }
    string|error updatedString = string:fromBytes(updatedBytes);
    if updatedString is string {
        return updatedString;
    }
    return originalString;
}
