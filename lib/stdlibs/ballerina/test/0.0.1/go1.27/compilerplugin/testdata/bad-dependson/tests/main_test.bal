import ballerina/test;

function helper() {
}

@test:Config {
    dependsOn: [helper]
}
function testMain() {
}
