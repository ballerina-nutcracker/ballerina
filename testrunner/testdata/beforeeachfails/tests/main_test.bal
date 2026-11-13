import ballerina/test;

@test:BeforeEach
function setupEach() {
    test:assertEquals(1, 2);
}

@test:Config {}
function testA() {
}

@test:Config {}
function testB() {
}
