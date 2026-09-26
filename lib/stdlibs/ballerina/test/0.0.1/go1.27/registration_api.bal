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

// This file has no jballerina equivalent and is TEMPORARY — see TODO.md's
// `typeof` entry. jballerina reads @test:* annotation field values at runtime
// via `(typeof f).@Config` (annotation_processor.bal), which this interpreter
// doesn't support yet. Until it does, the Go CLI's compile-time discovery step
// (cli/internal/testdiscovery) reads those same field values statically from
// the AST, and generated glue code (cli/internal/testglue) calls the functions
// below with the values already resolved as literal arguments.
//
// This is a stopgap, not a permanent design: once `typeof` lands,
// annotation_processor.bal must be ported for real and this file deleted, so
// `bal test` is driven by runtime introspection like every other stdlib in
// this port, not a bespoke compile-time special case. Do not build anything
// else on top of this file's existence lasting long-term.
//

# Registers a discovered `@test:Config` function. Intended for use by generated
# glue code only.
#
# jballerina's processConfigAnnotation gates the effective `enable` on
# --groups/--disable-groups/--tests filtering (filterGroups/filterDisableGroups/
# hasTest, all in filter.bal) on top of the raw `@test:Config.enable` field —
# reproduced here now that filter.bal exists (P8.7). Skips
# processConfigAnnotation's isolated-safety/serial-execution-reason computation
# entirely: that machinery only matters for choosing parallel vs. serial
# execution, which is out of scope for v1 (see TODO.md's start/wait entry) —
# `serialExecution` here is just the literal `@test:Config.serialExecution`
# field, same as before.
#
# + name - test function name
# + executableFunction - the test function itself
# + enable - the `enable` field of `@test:Config`
# + groups - the `groups` field of `@test:Config`
# + dependsOn - the `dependsOn` field of `@test:Config`
# + before - the `before` field of `@test:Config`, or `()` if absent
# + after - the `after` field of `@test:Config`, or `()` if absent
# + serialExecution - the `serialExecution` field of `@test:Config`
# + dataProvider - the `dataProvider` field of `@test:Config`, or `()` if absent
public function registerTestConfig(string name, function executableFunction, boolean enable,
        string[] groups, function[] dependsOn, function? before, function? after, boolean serialExecution,
        function? dataProvider) {
    boolean effectiveEnable = enable
            && (filterGroups.length() == 0 || hasGroup(groups, filterGroups))
            && (filterDisableGroups.length() == 0 || !hasGroup(groups, filterDisableGroups))
            && hasTest(name);
    foreach string 'group in groups {
        groupStatusRegistry.incrementTotalTest('group, effectiveEnable);
    }

    DataProviderReturnType? params = ();
    error? diagnostics = ();
    if dataProvider is function {
        any|error output = trap invokeFunction(dataProvider, []);
        if output is error {
            diagnostics = error("Failed to execute the data provider");
        } else {
            // DataProviderReturnType = error|map<AnyOrError[]>|AnyOrError[][] —
            // output's static type here is `any|error`, so this cast mirrors
            // jballerina's own `<DataProviderReturnType>providerOutput`.
            params = <DataProviderReturnType>output;
        }
    }
    dataDrivenTestParams[name] = params;

    testRegistry.addFunction(name = name, executableFunction = executableFunction, before = before,
            after = after, groups = groups, dependsOn = dependsOn, diagnostics = diagnostics,
            serialExecution = serialExecution, config = {enable: enable});
    executionManager.createTestFunctionMetaData(functionName = name, dependsOnCount = dependsOn.length(),
            enabled = effectiveEnable);
}

# Registers a discovered `@test:BeforeSuite` function.
public function registerBeforeSuite(string name, function executableFunction) {
    beforeSuiteRegistry.addFunction(name = name, executableFunction = executableFunction);
}

# Registers a discovered `@test:AfterSuite` function.
public function registerAfterSuite(string name, function executableFunction, boolean alwaysRun) {
    afterSuiteRegistry.addFunction(name = name, executableFunction = executableFunction, alwaysRun = alwaysRun);
}

# Registers a discovered `@test:BeforeEach` function.
public function registerBeforeEach(string name, function executableFunction) {
    beforeEachRegistry.addFunction(name = name, executableFunction = executableFunction);
}

# Registers a discovered `@test:AfterEach` function.
public function registerAfterEach(string name, function executableFunction) {
    afterEachRegistry.addFunction(name = name, executableFunction = executableFunction);
}

# Registers a discovered `@test:BeforeGroups` function.
public function registerBeforeGroups(string name, function executableFunction, string[] groups) {
    foreach string 'group in groups {
        beforeGroupsRegistry.addFunction('group, name = name, executableFunction = executableFunction);
    }
}

# Registers a discovered `@test:AfterGroups` function.
public function registerAfterGroups(string name, function executableFunction, string[] groups, boolean alwaysRun) {
    foreach string 'group in groups {
        afterGroupsRegistry.addFunction('group, name = name, executableFunction = executableFunction,
                alwaysRun = alwaysRun);
    }
}
