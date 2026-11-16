import ballerina/test;

@test:Config {}
function testPass() {
    test:assertTrue(true);
}
