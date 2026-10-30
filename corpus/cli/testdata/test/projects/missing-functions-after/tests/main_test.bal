import ballerina/test;

@test:Config {
    after: afterFuncNonExist
}
public function afterFuncNegative() {
}
