import ballerina/test;

boolean testABodyRan = false;

@test:Config {
    before: beforeTestA
}
function testA() {
    testABodyRan = true;
}

function beforeTestA() {
    test:assertEquals(1, 2);
}

@test:Config {}
function testZCheckSideEffect() {
    test:assertEquals(testABodyRan, false);
}

@test:Config {
    dependsOn: [testA]
}
function testB() {
}
