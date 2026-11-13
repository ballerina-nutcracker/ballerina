import ballerina/test;

@test:Config {}
function testSetup() {
}

@test:Config {
    groups: ["unit", "fast"],
    dependsOn: [testSetup]
}
function testMain() {
}

@test:Config {
    enable: false
}
function testDisabled() {
}

@test:BeforeSuite
function beforeAll() {
}

@test:AfterSuite {
    alwaysRun: true
}
function afterAll() {
}

@test:BeforeGroups {
    value: ["unit"]
}
function beforeUnitGroup() {
}

@test:AfterGroups {
    value: ["unit"],
    alwaysRun: true
}
function afterUnitGroup() {
}

@test:BeforeEach
function beforeEachTest() {
}

@test:AfterEach
function afterEachTest() {
}
