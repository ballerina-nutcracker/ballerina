import ballerina/test;

@test:Config {
    dataProvider: dataGen,
    runs: 2,
    minPassRate: 0.5
}
function testEvalDDHardPanic(int v) {
    if v == 2 {
        panic error("boom");
    }
    test:assertTrue(true);
}

function dataGen() returns map<[int]> {
    return {"a": [1], "b": [2]};
}
