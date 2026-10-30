import ballerina/test;

@test:AfterEach
function afterEachFails() {
    test:assertFail("afterEach failed");
}

@test:Config {}
function testA() {
}

@test:Config {}
function testB() {
}
