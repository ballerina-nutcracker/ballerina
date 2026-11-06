import ballerina/test;

int i = 0;

function init() {
    i = 1;
}

@test:Config {
    before: init
}
function testBefore() {
    test:assertTrue(i == 1, msg = "Expected i to be 1, but i = " + i.toString());
}
