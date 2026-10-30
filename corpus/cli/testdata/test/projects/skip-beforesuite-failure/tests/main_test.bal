import ballerina/test;

int afterSuiteCalls = 0;

@test:BeforeSuite
function beforeSuiteFails() {
    test:assertFail("beforeSuite failed");
}

@test:AfterSuite {
    alwaysRun: true
}
function afterSuiteAlwaysRuns() {
    afterSuiteCalls += 1;
}

@test:Config {}
function testA() {
}
