import ballerina/test;

@test:Config {
    dataProvider: tupleDataSet
}
function testAdd(int a, int b, int total) {
    test:assertEquals(a + b, total);
}

function tupleDataSet() returns [int, int, int][] {
    return [
        [1, 1, 2],
        [2, 2, 5]
    ];
}
