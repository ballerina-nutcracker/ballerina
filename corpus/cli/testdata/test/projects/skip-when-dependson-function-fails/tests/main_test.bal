import ballerina/test;

int j = 0;

@test:Config {}
public function test1() {
    j = j+1;
}

@test:Config {
    dependsOn: [test1]
}
public function test2() {
    int _ = 12/0;
}

@test:Config {
    dependsOn: [test2]
}
public function test3() {
    j = j+1;
}

@test:Config {
    dependsOn: [test3]
}
public function test4() {
    j = j+1;
}

@test:Config {}
public function test5() {
    j = j+1;
    test:assertEquals(j, 2);
}
