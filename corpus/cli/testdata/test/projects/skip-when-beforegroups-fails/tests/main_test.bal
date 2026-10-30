import ballerina/test;

string a = "";

@test:BeforeGroups { value : ["g1"] }
function beforeGroupsFunc1() {
    a += "2";
}

@test:BeforeGroups { value : ["g2"] }
function beforeGroupsFunc2() {
    int _ = 2/0;
}

@test:AfterGroups { value : ["g1", "g2"] }
function afterGroupsFunc1() {
    a += "7";
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
    a += "3";
}

@test:Config {
    groups : ["g2"],
    dependsOn: [testFunction2]
}
function testFunction3() {
    a += "5";
}

@test:Config {
    groups : ["g1", "g2"],
    dependsOn: [testFunction3]
}
function testFunction4() {
    a += "6";
}

@test:Config {
    groups : ["g2"],
    dependsOn: [testFunction4]
}
function testFunction5() {
    a += "8";
}
