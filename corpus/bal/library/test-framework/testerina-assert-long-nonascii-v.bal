import ballerina/io;
import ballerina/test;

// A 3-byte multi-byte character straddling getFormattedString's 80-character
// chunking boundary, past maxArgLength so the value actually gets rewrapped.
final string longNonAsciiValue = "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX" +
        "字XXXXXXXXXX";

function callAssertLongNonAscii() returns error? {
    test:assertEquals(1, longNonAsciiValue);
}

public function main() {
    error? failure = trap callAssertLongNonAscii();
    if failure is error {
        // Message length changes by 2 runes if chunking falls back to byte-based boundaries.
        io:println("messageLength: ", failure.message().length());
    }
    // @output messageLength: 149
}
