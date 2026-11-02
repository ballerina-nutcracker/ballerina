import ballerina/test;

function setupFails() {
    test:assertFail("setup always fails");
}

int invokeCount = 0;

@test:Config {
    before: setupFails,
    runs: 3,
    minPassRate: 0.5
}
function testEvalSkipped() {
    invokeCount += 1;
}

@test:Config {
    dependsOn: [testEvalSkipped]
}
function testDependent() {
}
