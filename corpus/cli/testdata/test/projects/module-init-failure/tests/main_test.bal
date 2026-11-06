import ballerina/test;

int a = 1 / 0;

@test:Config {}
function testFunc() {
    test:assertEquals(a, 0);
}
