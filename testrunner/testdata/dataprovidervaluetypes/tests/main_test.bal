import ballerina/test;

function errorDataSet() returns map<[error, string]> {
    error e = error("foo");
    return {"foo": [e, "foo"]};
}

function getMessage(error input) returns string {
    return input.message();
}

@test:Config {
    dataProvider: errorDataSet
}
function testErrorData(error input, string expected) {
    test:assertEquals(getMessage(input), expected);
}

function isEven(int x) returns boolean {
    return x % 2 == 0;
}

function functionDataSet() returns map<[function (int) returns boolean, int]> {
    return {"even": [isEven, 4]};
}

@test:Config {
    dataProvider: functionDataSet
}
function testFunctionData(function (int) returns boolean predicate, int value) {
    test:assertTrue(predicate(value));
}
