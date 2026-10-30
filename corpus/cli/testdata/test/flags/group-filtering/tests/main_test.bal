import ballerina/test;

@test:Config {
    groups: ["unit"]
}
function testUnit() {
    test:assertTrue(true);
}

@test:Config {
    groups: ["integration"]
}
function testIntegration() {
    test:assertTrue(true);
}
