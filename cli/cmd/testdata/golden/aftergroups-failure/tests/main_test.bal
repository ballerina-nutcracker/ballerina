import ballerina/test;

@test:AfterGroups {
    value: ["unit"]
}
function afterUnitGroupFails() {
    test:assertFail("afterGroups failed");
}

@test:Config {
    groups: ["unit"]
}
function testInGroup() {
}
