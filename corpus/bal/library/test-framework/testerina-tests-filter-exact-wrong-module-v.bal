import ballerina/io;
import ballerina/test;

int fooTestRan = 0;
int barTestRan = 0;

function fooTest() {
    fooTestRan += 1;
}

function barTest() {
    barTestRan += 1;
}

public function main() {
    // An exact entry for a different module ("combinedqualmod.sub") must not shadow the
    // wildcard "combinedqualmod:foo*" that matches fooTest in the running module.
    test:setTestOptions("target", "combinedqualmod", "combinedqualmod", "false", "false", "", "",
            "combinedqualmod.sub:fooTest,combinedqualmod:foo*", "false", "false");
    test:registerTestConfig("fooTest", fooTest, true, [], [], (), (), false, ());
    test:registerTestConfig("barTest", barTest, true, [], [], (), (), false, ());

    int exitCode = test:startSuite();

    io:println("exitCode: ", exitCode);
    io:println("fooTestRan: ", fooTestRan);
    io:println("barTestRan: ", barTestRan);
    // @output 		[pass] fooTest
    // @output
    // @output
    // @output 		1 passing
    // @output 		0 failing
    // @output 		0 skipped
    // @output
    // @output 		Test execution time : <DURATION>s
    // @output exitCode: 0
    // @output fooTestRan: 1
    // @output barTestRan: 0
}
