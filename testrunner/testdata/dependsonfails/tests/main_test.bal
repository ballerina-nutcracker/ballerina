import ballerina/test;

@test:Config {}
function testA() {
    test:assertEquals(1, 2);
}

@test:Config {
    dependsOn: [testA]
}
function testB() {
}
