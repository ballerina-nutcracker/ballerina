import ballerina/test;

@test:Config {dependsOn: [testB]}
function testA() {
    test:assertTrue(true);
}

@test:Config {dependsOn: [testA]}
function testB() {
    test:assertTrue(true);
}
