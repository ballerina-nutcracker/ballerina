import ballerina/test;

@test:Config {}
function testGood() {
    test:assertTrue(true);
}

@test:Config {}
function testBad() {
    test:assertTrue(false, "boom");
}
