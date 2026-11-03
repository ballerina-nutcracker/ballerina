import ballerina/test;

@test:Config {}
function testBad() {
    test:assertTrue(false, "boom");
}
