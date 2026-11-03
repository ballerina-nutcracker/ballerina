import ballerina/test;

int c = 0;

@test:Config {runs: 3, minPassRate: 0.6}
function testEval() {
    c += 1;
    test:assertTrue(c != 2, msg = "run " + c.toString());
}

int d = 0;

@test:Config {runs: 3, minPassRate: 0.9}
function testEvalFails() {
    d += 1;
    test:assertTrue(d == 2, msg = "run " + d.toString());
}
