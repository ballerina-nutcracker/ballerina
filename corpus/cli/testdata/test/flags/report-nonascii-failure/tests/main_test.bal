import ballerina/test;

@test:Config {}
function testFail() {
    test:assertFail("café says \"hi\"");
}
