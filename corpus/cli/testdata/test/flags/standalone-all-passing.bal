import ballerina/io;
import ballerina/test;

public function main() {
    io:println("MAIN_RAN_UNEXPECTEDLY");
}

@test:Config {}
function testOne() {
    test:assertTrue(true);
}
