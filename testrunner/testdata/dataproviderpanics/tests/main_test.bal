import ballerina/test;

int beforeEachCount = 0;

@test:BeforeEach
function beforeEach() {
    beforeEachCount += 1;
}

@test:Config {
    dataProvider: badDataSet
}
function testProviderFails(int x) {
    test:assertEquals(x, x);
}

function badDataSet() returns map<[int]> {
    test:assertEquals(1, 2);
    return {};
}

@test:Config {}
function testZNormal() {
    test:assertEquals(beforeEachCount, 1);
}
