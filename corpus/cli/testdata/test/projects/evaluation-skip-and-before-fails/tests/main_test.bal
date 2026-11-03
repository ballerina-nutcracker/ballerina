import ballerina/test;

@test:Config {}
function testA() {
    test:assertFail("a fails");
}

@test:Config {dependsOn: [testA], runs: 2, minPassRate: 0.5}
function testEvalSkippedByDependsOn() {
}

function setupFails() {
    test:assertFail("setup always fails");
}

@test:Config {before: setupFails, runs: 3, minPassRate: 0.5}
function testEvalBeforeFails() {
}
