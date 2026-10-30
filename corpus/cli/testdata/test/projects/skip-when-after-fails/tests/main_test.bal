import ballerina/test;

string a = "before";

@test:AfterEach
public function afterEach() {
    a = a + "afterEach";
}

public function afterFunc() {
    int _ = 12/0;
}

@test:Config {
    after: afterFunc
}
public function test1() {
    a = a + "test";
}

@test:Config {
    dependsOn: [test1]
}
public function test2() {
    a = a + "test2";
}

@test:Config {}
public function test3() {
    a = a + "test";
}
