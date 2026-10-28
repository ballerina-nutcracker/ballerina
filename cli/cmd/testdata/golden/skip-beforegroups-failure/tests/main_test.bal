import ballerina/test;

@test:BeforeGroups {
    value: ["unit"]
}
function beforeUnitGroupFails() {
    test:assertFail("beforeGroups failed");
}

@test:Config {
    groups: ["unit"]
}
function testInGroup() {
}
