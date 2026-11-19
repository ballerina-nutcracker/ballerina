import ballerina/io;
import ballerina/test;

@test:Config {}
function testLib() {
    io:println("in testLib");
    test:assertEquals(f(), 1);
}
