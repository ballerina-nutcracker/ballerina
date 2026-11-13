import ballerina/test;

@test:Config {
    dataProvider: squareDataSet
}
function testSquare(int x, int y) {
    test:assertEquals(x * x, y);
}

function squareDataSet() returns map<[int, int]> {
    return {"one": [1, 1], "two": [2, 4]};
}

@test:Config {
    dependsOn: [testSquare]
}
function testUsesAll() {
}
