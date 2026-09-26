import ballerina/io;
import ballerina/test;

int testFooRan = 0;
int otherBazRan = 0;
int unrelatedQuxRan = 0;

function testFooAlpha() {
    testFooRan += 1;
}

function otherBazBeta() {
    otherBazRan += 1;
}

function unrelatedQux() {
    unrelatedQuxRan += 1;
}

public function main() {
    // --tests "testFoo*,otherBaz*" has two wildcard patterns matching two
    // disjoint sets of functions. filter.bal#hasTest must check every
    // wildcard filter, not stop after the first one — otherwise
    // otherBazBeta would never get a chance to match.
    test:setTestOptions("target", "multiwildcardfiltermod", "multiwildcardfiltermod", "false", "false", "", "",
            "testFoo*,otherBaz*", "false", "false");
    test:registerTestConfig("testFooAlpha", testFooAlpha, true, [], [], (), (), false, ());
    test:registerTestConfig("otherBazBeta", otherBazBeta, true, [], [], (), (), false, ());
    test:registerTestConfig("unrelatedQux", unrelatedQux, true, [], [], (), (), false, ());

    int exitCode = test:startSuite();

    io:println("exitCode: ", exitCode);
    io:println("testFooRan: ", testFooRan);
    io:println("otherBazRan: ", otherBazRan);
    io:println("unrelatedQuxRan: ", unrelatedQuxRan);
    // @output 		[pass] otherBazBeta
    // @output 		[pass] testFooAlpha
    // @output
    // @output
    // @output 		2 passing
    // @output 		0 failing
    // @output 		0 skipped
    // @output
    // @output 		Test execution time : <DURATION>s
    // @output exitCode: 0
    // @output testFooRan: 1
    // @output otherBazRan: 1
    // @output unrelatedQuxRan: 0
}
