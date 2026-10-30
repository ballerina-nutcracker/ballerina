import ballerina/test;

@test:Config {
    before: beforeFuncNonExist
}
public function beforeFuncNegative() {
}
