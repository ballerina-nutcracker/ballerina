import ballerina/test;

string testString = "";

@test:BeforeEach
function beforeEachFunc() {
    testString += "beforeEach";
}

@test:Config {}
public function testFunc() {
    testString += "test";
}

@test:AfterEach
public function afterEachFunc() {
    testString += "afterEach";
}

@test:Config {
    dependsOn: [testFunc]
}
public function testFunc2() {
    test:assertEquals(testString, "beforeEachtestafterEachbeforeEach");
}

@test:Config {
    dependsOn: [testFunc2]
}
public function testFunc3() {
    test:assertEquals(testString, "beforeEachtestafterEachbeforeEachafterEachbeforeEach");
}
