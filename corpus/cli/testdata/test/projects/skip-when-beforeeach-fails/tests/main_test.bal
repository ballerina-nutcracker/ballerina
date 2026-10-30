import ballerina/test;

string a = "before";

@test:BeforeEach
public function beforeEach() {
    int _ = 12/0;
}

@test:AfterEach
public function afterEachHook() {
    a = a + "afterEach";
}

@test:Config {}
public function test1() {
    a = a + "test1";
}

@test:Config {}
public function test2() {
    a = a + "test2";
}

@test:Config {}
public function test3() {
    a = a + "test3";
}
