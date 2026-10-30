import ballerina/test;

@test:Config {
    after: afterTestA
}
function testA() {
}

function afterTestA() {
    test:assertEquals(1, 2);
}

@test:Config {
    dependsOn: [testA]
}
function testB() {
}
