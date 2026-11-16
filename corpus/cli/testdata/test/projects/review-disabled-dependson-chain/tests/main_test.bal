import ballerina/test;

@test:Config {enable: false}
function testA() {
}

@test:Config {enable: false, dependsOn: [testA]}
function testB() {
}

@test:Config {}
function testC() {
    test:assertTrue(true);
}
