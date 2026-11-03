import ballerina/test;

function squareDataSet() returns map<[int, int]> {
    return {
        "good": [2, 4],
        "bad": [3, 999]
    };
}

@test:Config {dataProvider: squareDataSet, runs: 2, minPassRate: 0.5}
function testDataDrivenCrossesThreshold(int x, int y) {
    test:assertEquals(x * x, y);
}

@test:Config {dataProvider: squareDataSet, runs: 2, minPassRate: 0.9}
function testDataDrivenBelowThreshold(int x, int y) {
    test:assertEquals(x * x, y);
}
