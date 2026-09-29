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
    // filterKeyBasedTests must accumulate both data-provider sub-keys, not overwrite the first.
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
