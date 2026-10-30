import ballerina/test;

@test:Config {
    dataProvider: emptyDataSet
}
function testSquare(int input, int expected) {
    test:assertEquals(input * input, expected);
}

function emptyDataSet() returns map<[int, int]> {
    return {};
}
