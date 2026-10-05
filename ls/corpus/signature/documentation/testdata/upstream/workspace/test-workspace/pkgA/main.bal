import test/pkgB;

public function main() {
    // Test 1: Cross-package function call - processData (at opening parenthesis)
    string result1 = pkgB:processData("test", 3);

    // Test 2: Cross-package function call - calculate (between parameters)
    float result2 = pkgB:calculate(10.5, 20.3, "add");

    // Test 3: Cross-package method call - DataProcessor
    pkgB:DataProcessor processor = new("myProcessor");
    string result3 = processor.process("hello", true);

    // Test 4: Cross-package method call - DataProcessor transform
    string result4 = processor.transform("world", "[", "]");

    // Test 5: Cross-package action call - ApiClient fetch
    pkgB:ApiClient cl = new("https://api.example.com");
    string|error result5 = cl->fetch(123, false);

    // Test 6: Cross-package action call - ApiClient send
    map<string> headers = {"Authorization": "Bearer token"};
    boolean|error result6 = cl->send("POST", "/submit", headers);
}
