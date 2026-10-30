import ballerina/test;

@test:Config {
    dataProvider: dataGen
}
function stringDataProviderTest(string fValue, string sValue, string result) returns error? {
    int value1 = check 'int:fromString(fValue);
    int value2 = check 'int:fromString(sValue);
    int result1 = check 'int:fromString(result);
    test:assertEquals(value1 + value2, result1, msg = "The sum is not correct");
    return;
}

function dataGen() returns (string[][]) {
    return [["1", "2", "3"], ["10", "20", "30"], ["5", "6", "11"]];
}

@test:Config {
    dataProvider: dataGen2
}
function stringDataProviderTest2(string fValue, string sValue, string result) returns error? {
    int value1 = check 'int:fromString(fValue);
    int value2 = check 'int:fromString(sValue);
    int result1 = check 'int:fromString(result);
    test:assertEquals(value1 + value2, result1, msg = "The sum is not correct");
    return;
}

function dataGen2() returns (string[][]) {
    return [["1", "2", "3"]];
}

@test:Config {
    dataProvider: dataGen3
}
function jsonDataProviderTest(json fValue, json sValue, json result) {
    json a = {"a": "a"};
    json b = {"b": "b"};
    json c = {"c": "c"};
    test:assertEquals(fValue, a, msg = "json data provider failed");
    test:assertEquals(sValue, b, msg = "json data provider failed");
    test:assertEquals(result, c, msg = "json data provider failed");
}

function dataGen3() returns (json[][]) {
    return [[{"a": "a"}, {"b": "b"}, {"c": "c"}]];
}
