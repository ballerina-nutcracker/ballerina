import ballerina/test;

string a = "";

@test:Config{
    groups: ["g1","g2"]
}
function testFunc1 () {
    a += "1";
    test:assertFalse(false, msg = "errorMessage");
}

@test:Config{
    groups: ["g1","g2"],
    dependsOn: [testFunc1]
}
function testFunc2 () {
    a += "2";
    test:assertFalse(false, msg = "errorMessage");
}

@test:Config{
    groups: ["g1"],
    dependsOn: [testFunc2]
}
function testFunc3 () {
    a += "3";
    test:assertFalse(false, msg = "errorMessage");
}

@test:Config{
    enable : false,
    groups: ["g1"],
    dependsOn: [testFunc3]
}
function testFunc5 () {
    a += "4";
    test:assertFalse(false, msg = "errorMessage");
}

@test:AfterGroups { value : ["g1"] }
function afterGroupsFunc1() {
    a += "5";
}
@test:AfterSuite {}
function afterSuiteFunc() {
    test:assertEquals(a, "1235");
}
