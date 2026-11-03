import ballerina/test;

@test:Config {}
function testReturnsError() returns error? {
    return error("boom");
}

@test:Config {}
function testPanicsNonAssertion() {
    panic error("kaboom");
}

@test:Config {}
function testAssertionStillHasColon() {
    test:assertTrue(false, "boom");
}
