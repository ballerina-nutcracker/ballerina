import ballerina/test;

@test:BeforeSuite
function beforeSuiteFails() {
    test:assertFail("beforeSuite failed");
}

@test:AfterSuite {
    alwaysRun: true
}
function afterSuiteAlwaysRuns() {
    test:assertFail("afterSuite ran");
}

@test:Config {}
function testA() {
}
