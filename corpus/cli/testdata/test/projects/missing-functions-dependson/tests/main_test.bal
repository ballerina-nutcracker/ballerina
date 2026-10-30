import ballerina/test;

@test:Config {
    dependsOn: [nonExisting]
}
public function test2() {
}
