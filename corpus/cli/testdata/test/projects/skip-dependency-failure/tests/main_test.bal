import ballerina/test;

@test:Config {}
function testDependency() {
    test:assertFail("dependency setup failed");
}

@test:Config {
    dependsOn: [testDependency]
}
function testDependent() {
}
