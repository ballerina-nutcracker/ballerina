import ballerina/test;

@test:Config {}
function testPass() {
    test:assertTrue(true);
}

@test:Config {}
function testFail() {
    test:assertTrue(false, "boom");
}
