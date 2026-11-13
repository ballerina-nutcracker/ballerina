import ballerina/test;

// This test should not run
@test:Config{
    enable:false //test comment
}
function testDisableFunc1 () {
    test:assertTrue(false, msg = "this test is not expected to run");
}

// This test should run
@test:Config{
    enable:true
}
function testDisableFunc2 () {
    test:assertTrue(true);
}

// Test without enable attribute. This should run.
@test:Config{
}
function testDisableFunc3 () {
    test:assertFalse(false);
}

@test:Config {
    "enable": false
}
function testDisableFunc4() {
    test:assertTrue(false, msg = "this test is not expected to run");
}

@test:Config {
    // this is an intentional comment
    enable: false
}
function testDisableFunc5() {
    test:assertTrue(false, msg = "this test is not expected to run");
}

@ test:Config { // space after @ is intentional
    enable: false
}
function testDisableFunc6() {
    test:assertTrue(false, msg = "this test is not expected to run");
}
