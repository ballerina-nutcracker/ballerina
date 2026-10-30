import ballerina/test;

@test:Config {}
function testOutsidePath() {
    test:assertFail("must never run - declared outside tests/");
}

public function main() {
}
