import ballerina/io;

public function main() {
    string b64 = "SGVsbG8=";
    var decoded = array:fromBase64(b64);
    io:println(decoded); // @output [72,101,108,108,111]

    string hex = "48656c6c6f";
    var decodedHex = array:fromBase16(hex);
    io:println(decodedHex); // @output [72,101,108,108,111]

    // error cases
    string badB64 = "SGVsbG8--";
    var decodedBad = array:fromBase64(badB64);
    io:println(decodedBad); // @output error("failed to decode base64 string")

    string badHex = "48656c6c6f==";
    var decodedBadHex = array:fromBase16(badHex);
    io:println(decodedBadHex); // @output error("failed to decode base16 string")
}
