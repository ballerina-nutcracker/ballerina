import ballerina/test;

string testStr = "";

function before() {
    testStr = testStr + "before";
}

@test:Config {
    dependsOn: [testWithBefore1]
}
public function testWithBefore2() {
    testStr = testStr + "test2";
}

@test:Config {
    before: before,
    dependsOn: [test4] // added to preserve order of test execution
}
public function testWithBefore1() {
    testStr = testStr + "test1";
}

@test:Config {
    dependsOn: [testWithBefore2]
}
public function testWithBefore3() {
    testStr = testStr + "test3";
}

@test:Config {
    dependsOn: [testWithBefore3]
}
public function testWithBefore4() {
    test:assertEquals(testStr, "beforetest1test2test3", msg = "Order is not correct");
}
