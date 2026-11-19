import ballerina/test;
import testorg/liba;

@test:Config {}
function testApp() {
    test:assertEquals(liba:f(), 1);
}
