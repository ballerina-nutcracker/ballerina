import ballerina/test;

string a = "";

@test:BeforeSuite
function beforeSuiteFunc() {
    a += "1";
}

@test:BeforeGroups { value : ["g1"] }
function beforeGroupsFunc1() {
    a += "7";
}

@test:AfterGroups { value : ["g1"] }
function afterGroupsFunc1() {
    a += "9";
}

@test:BeforeEach
function beforeEachFunc() {
    a += "2";
}

@test:AfterEach
function afterEachFunc() {
    a += "6";
}

function beforeFunc() {
    a += "3";
}

@test:Config {
    before: beforeFunc,
    after: afterFunc
}
function testFunction() {
    a += "4";
}

@test:Config {
    groups: ["g1"],
    dependsOn: [testFunction]
}
function testFunction2() {
    a += "8";
}

function afterFunc() {
    a += "5";
}

@test:AfterSuite {}
function afterSuiteFunc() {
    test:assertEquals(a, "12345672869");
}
