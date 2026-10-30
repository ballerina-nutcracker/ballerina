import ballerina/test;

@test:BeforeSuite
function setupSuite() {
    test:assertEquals(1, 2);
}

@test:AfterSuite {}
function teardownNormal() {
    test:assertEquals(1, 2);
}

@test:AfterSuite {
    alwaysRun: true
}
function teardownAlways() {
    test:assertEquals(1, 2);
}

@test:Config {}
function testA() {
}
