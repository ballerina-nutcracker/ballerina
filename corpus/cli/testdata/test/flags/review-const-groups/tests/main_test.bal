import ballerina/test;

const string[] GROUPS = ["slow"];

@test:Config {groups: GROUPS}
function testSlow() {
}

@test:Config {groups: ["fast"]}
function testFast() {
}
