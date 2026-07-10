import ballerina/io;

public function main() {
    string b64 = "SGVsbG8=";
    var decoded = array:fromBase64(b64);
    io:println(decoded);

    string hex = "48656c6c6f";
    var decodedHex = array:fromBase16(hex);
    io:println(decodedHex);
}
