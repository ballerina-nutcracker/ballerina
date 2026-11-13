import ballerina/test;

string testString = "";

@test:Config {
    dependsOn: [test1]
}
public function test2() {
    testString = testString + "test2";
}

@test:Config {}
public function test1() {
    testString = testString + "test1";
}

@test:Config {
    dependsOn: [test2]
}
public function test3() {
    testString = testString + "test3";
}

@test:Config {
    dependsOn: [test3]
}
public function test4() {
    test:assertEquals(testString, "test1test2test3", msg = "Order is not correct");
}
