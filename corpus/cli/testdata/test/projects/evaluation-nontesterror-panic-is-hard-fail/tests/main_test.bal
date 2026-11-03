import ballerina/test;

int counter = 0;

@test:Config {
    runs: 3,
    minPassRate: 0.5
}
function testEvalHardPanic() {
    counter += 1;
    if counter == 2 {
        panic error("boom");
    }
    test:assertTrue(true);
}
