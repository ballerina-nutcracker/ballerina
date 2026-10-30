import ballerina/test;

function dataSet() returns map<[int, int]> {
    return {"good": [2, 4], "bad": [2, 5]};
}

@test:Config {
    dataProvider: dataSet
}
function testSquare(int input, int expected) {
    test:assertEquals(input * input, expected);
}
