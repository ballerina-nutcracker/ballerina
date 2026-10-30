import ballerina/test;

public isolated client class TestClient {
    remote function getMessage() returns string {
      return "Test Client";
    }
}

@test:Config {}
public function testAsyncRemoteFunctionMain() returns (error?) {
    MainClient 'client = new MainClient();
    future<string> message = start 'client->getMessage();
    string|error msg1 = wait message;
    test:assertEquals(msg1, "Main Client");
}

@test:Config {}
public function testAsyncRemoteFunctionTest() returns (error?) {
    TestClient 'client = new TestClient();
    future<string> message = start 'client->getMessage();
    string|error msg1 = wait message;
    test:assertEquals(msg1, "Test Client");
}
