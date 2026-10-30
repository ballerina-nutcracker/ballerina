import ballerina/test;

string a = "";

@test:BeforeGroups { value : ["g1"] }
function beforeGroupsFunc1() {
    panic error("Failed before group for g1");
}

@test:BeforeGroups { value : ["g2"] }
function beforeGroupsFunc2() {
    a += "3";
}

@test:AfterGroups { value : ["g1"], alwaysRun: true }
function afterGroupsFunc1() {
    a += "6";
}

@test:AfterGroups { value : ["g1"] }
function afterGroupsFunc2() {
    a += "7";
}

@test:AfterGroups { value : ["g1", "g2"] }
function afterGroupsFunc3() {
    a += "9";
}

@test:Config {}
function testFunction() {
    a += "1";
}

@test:Config {
    groups: ["g1"],
    dependsOn: [testFunction]
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
    groups : ["g1", "g2"],
    dependsOn: [testFunction]
}
function testFunction4() {
    a += "5";
}

@test:Config {
    groups : ["g2"],
    dependsOn: [testFunction4]
}
function testFunction5() {
    a += "8";
}

@test:AfterSuite {}
function afterSuiteFunc() {
    test:assertEquals(a, "135689");
}
