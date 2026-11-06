import ballerina/test;

@test:Config {}
function testMapKeyMismatch() {
    map<anydata> actual = {name: "Alice", age: 30};
    map<anydata> expected = {name: "Alice", city: "NYC"};
    test:assertEquals(actual, expected);
}
