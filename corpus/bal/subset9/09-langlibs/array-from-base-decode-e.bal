import ballerina/io;

public function main() {
    string badB64 = "SGVsbG8--";
    var decoded = array:fromBase64(badB64);
    io:println(decoded); // @output error("failed to decode base64 string")

    string badHex = "48656c6c6f==";
    var decodedHex = array:fromBase16(badHex);
    io:println(decodedHex); // @output error("failed to decode base16 string")
}
