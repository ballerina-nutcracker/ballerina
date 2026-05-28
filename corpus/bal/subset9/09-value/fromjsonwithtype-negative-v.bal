import ballerina/io;

type IntArray int[];
type StringArray string[];
type Person record {|
    string name;
    int age;
|};
type Closed record {|
    string x;
|};

public function main() {
    checkpanic run();
}

function run() returns error? {
    json badBool = "2022";
    io:println(badBool.fromJsonWithType(boolean) is error); // @output true

    json badArr = ["a", "b"];
    io:println(badArr.fromJsonWithType(IntArray) is error); // @output true

    json badNumArr = [1, 2];
    io:println(badNumArr.fromJsonWithType(StringArray) is error); // @output true

    json badString = "foobar";
    io:println(badString.fromJsonWithType(int) is error); // @output true

    json missingField = {"name": "Alice"};
    io:println(missingField.fromJsonWithType(Person) is error); // @output true

    json extraField = {"x": "a", "y": 1};
    io:println(extraField.fromJsonWithType(Closed) is error); // @output true

    json nilVal = ();
    io:println(nilVal.fromJsonWithType(Person) is error); // @output true
    return;
}
