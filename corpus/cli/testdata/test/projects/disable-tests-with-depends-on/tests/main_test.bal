import ballerina/test;

@test:Config {}
function testFunction1() {
    test:assertTrue(true);
}

@test:Config {
    dependsOn: [testFunction1],
    enable: false
}
function testDisableFunction2() {
    test:assertTrue(false, "This test should not run");
}

@test:Config {
    dependsOn: [testDisableFunction2]
}
function testFunction3() {
    test:assertFalse(false, "This test should not run");
}
