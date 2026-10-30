import ballerina/test;

function mismatchedDataSet() returns map<[string, string]> {
    return {"one": ["not", "numbers"]};
}

@test:Config {
    dataProvider: mismatchedDataSet
}
function testMismatched(int a, int b) {
    int sum = a + b;
    test:assertTrue(sum >= 0);
}
