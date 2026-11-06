import ballerina/test;

function afterFails() {
    test:assertFail("after hook failed");
}

@test:Config {
    after: afterFails
}
function testA() {
}

@test:Config {
    dependsOn: [testA]
}
function testDependent() {
}
