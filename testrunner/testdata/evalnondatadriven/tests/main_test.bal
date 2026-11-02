import ballerina/test;

int crossesCounter = 0;

// Fails on odd-numbered calls, passes on even: 2/4 = 0.5, meeting the
// threshold exactly.
@test:Config {
    runs: 4,
    minPassRate: 0.5
}
function testCrossesThreshold() {
    crossesCounter += 1;
    test:assertTrue(crossesCounter % 2 == 0, msg = "call " + crossesCounter.toString() + " is odd");
}

int belowCounter = 0;

@test:Config {
    runs: 3,
    minPassRate: 0.9
}
function testBelowThreshold() {
    belowCounter += 1;
    test:assertFail("always fails, run " + belowCounter.toString());
}
