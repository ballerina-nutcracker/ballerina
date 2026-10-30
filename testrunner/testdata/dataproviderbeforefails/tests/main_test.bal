import ballerina/test;

@test:Config {
    dataProvider: dataSet,
    before: beforeRow
}
function testRows(int x) {
    test:assertEquals(x, x);
}

function beforeRow() {
    test:assertEquals(1, 2);
}

function dataSet() returns int[][] {
    return [[1], [2]];
}
