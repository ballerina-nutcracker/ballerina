import ballerina/test;

@test:Config {}
function testFunctionPass1() {
    test:assertTrue(true, msg = "Failed!");
}

@test:Config {}
function testFunctionPass2() {
    test:assertTrue(true, msg = "Failed!");
}

@test:Config {}
function testFunctionFail1() {
    test:assertTrue(false, msg = "Failed!");
}

@test:Config {}
function testFunctionFail2() {
    test:assertTrue(false, msg = "Failed!");
}
