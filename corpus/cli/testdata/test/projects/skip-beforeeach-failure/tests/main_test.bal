import ballerina/test;

int beforeEachCalls = 0;

@test:BeforeEach
function beforeEachFails() {
    beforeEachCalls += 1;
    test:assertFail("beforeEach failed");
}

@test:Config {}
function testA() {
}

@test:Config {}
function testB() {
}
