import ballerina/test;

@test:Config {}
function testGood() {
    test:assertTrue(true);
}
