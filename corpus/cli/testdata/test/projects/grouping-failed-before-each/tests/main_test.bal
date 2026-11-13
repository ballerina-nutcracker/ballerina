import ballerina/test;

string a = "";

@test:BeforeGroups { value : ["g1"] }
function beforeGroupsFunc1() {
    a += "1";
}

@test:BeforeGroups { value : ["g2"] }
function beforeGroupsFunc2() {
    a += "3";
}

@test:AfterGroups { value : ["g1", "g2"] }
function afterGroupsFunc1() {
    a += "7";
}

@test:BeforeEach
function beforeEachFunction() {
    test:assertFail("Before each function failed.");
}

@test:Config {
    groups: ["g1"]
}
function testFunction2() {
    a += "2";
}

@test:Config {
    groups : ["g2"],
    dependsOn: [testFunction2]
}
function testFunction3() {
    a += "4";
}

@test:Config {
    groups : ["g1", "g2"]
}
function testFunction4() {
    a += "5";
}

@test:Config {
    groups : ["g2"],
    dependsOn: [testFunction4]
}
function testFunction5() {
    a += "6";
}

@test:AfterSuite {}
function afterSuiteFunc() {
    test:assertEquals(a, "1");
}
