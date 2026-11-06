import ballerina/test;

@test:Config {}
function testInSubmod() {
    test:assertTrue(true);
}
