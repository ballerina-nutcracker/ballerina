import ballerina/test;

string a = "before";

@test:BeforeEach
public function beforeEach() {
    a = a + "beforeEach";
}

@test:AfterEach
public function afterEach() {
    int _ = 12/0;
}

@test:Config {}
public function test1() {
    a = a + "test";
}

@test:Config {}
public function test2() {
    a = a + "test";
}

@test:Config {}
public function test3() {
    a = a + "test";
}
