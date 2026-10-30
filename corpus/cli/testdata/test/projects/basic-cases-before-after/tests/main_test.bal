import ballerina/test;

string testString = "";

function beforeFunc() {
    testString += "before";
}

@test:Config {
    before: beforeFunc,
    after: afterFunc
}
public function testFunc() {
    testString += "test";
}

public function afterFunc() {
    testString += "after";
}

@test:Config {
    dependsOn: [testFunc]
}
public function testFunc2() {
    test:assertEquals(testString, "beforetestafter");
}
