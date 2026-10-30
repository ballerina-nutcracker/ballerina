import ballerina/test;

@test:Config {}
function testPasses() {
}

@test:Config {}
function testFails() {
    test:assertEquals(1, 2);
}
