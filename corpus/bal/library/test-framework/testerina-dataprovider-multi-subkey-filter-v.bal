import ballerina/io;
import ballerina/test;

int oneRan = 0;
int twoRan = 0;
int threeRan = 0;

function squareDataSet() returns map<[int, int]> {
    return {"one": [1, 1], "two": [2, 4], "three": [3, 9]};
}

function testSquare(int input, int expected) {
    if input == 1 {
        oneRan += 1;
    } else if input == 2 {
        twoRan += 1;
    } else {
        threeRan += 1;
    }
    test:assertEquals(input * input, expected);
}

public function main() {
    // --tests testSquare#one,testSquare#three selects two sub-keys of the
    // same data-driven test. filterKeyBasedTests must accumulate both
    // suffixes under the one "testSquare" entry, not have the second
    // overwrite the first (which would silently drop "one" and leave "two"
    // unfiltered-out instead of "three" included) — see filter.bal's
    // isFilterSubTestsContains, which must be keyed consistently with
    // getFilterSubTest/addFilterSubTest.
    test:setTestOptions("target", "ddmultisubkeymod", "ddmultisubkeymod", "false", "false", "", "",
            "testSquare#one,testSquare#three", "false", "false");
    test:registerTestConfig("testSquare", testSquare, true, [], [], (), (), false, squareDataSet);

    int exitCode = test:startSuite();

    io:println("exitCode: ", exitCode);
    io:println("oneRan: ", oneRan);
    io:println("twoRan: ", twoRan);
    io:println("threeRan: ", threeRan);
    // @output 		[pass] testSquare#one
    // @output 		[pass] testSquare#three
    // @output
    // @output
    // @output 		2 passing
    // @output 		0 failing
    // @output 		0 skipped
    // @output
    // @output 		Test execution time : <DURATION>s
    // @output exitCode: 0
    // @output oneRan: 1
    // @output twoRan: 0
    // @output threeRan: 1
}
