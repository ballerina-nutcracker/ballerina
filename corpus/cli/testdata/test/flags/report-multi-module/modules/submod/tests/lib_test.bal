import ballerina/test;

@test:Config {}
function testSubmodule() {
    test:assertEquals(double(2), 4);
}
