import ballerina/test;

function errorProvider() returns map<[int, int, int]>|error {
    return error("Error occurred while generating data set.");
}

@test:Config {
    dataProvider: errorProvider
}
function testWithErrorProvider(int a, int b, int result) {
    test:assertEquals(a + b, result);
}
