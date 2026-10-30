import ballerina/test;

@test:BeforeGroups {
    value: ["g"]
}
function setupGroup() {
    test:assertEquals(1, 2);
}

@test:AfterGroups {
    value: ["g"]
}
function teardownGroup() {
    test:assertEquals(1, 2);
}

@test:Config {
    groups: ["g"]
}
function testA() {
}

@test:Config {
    groups: ["g"]
}
function testB() {
}
