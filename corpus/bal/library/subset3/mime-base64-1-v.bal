import ballerina/io;
import ballerina/mime;

public function main() returns error? {
    string|byte[]|mime:EncodeError enc = mime:base64Encode("Hello");
    if enc is string {
        io:println(enc);
    }

    string|byte[]|mime:DecodeError dec = mime:base64Decode("SGVsbG8=");
    if dec is string {
        io:println(dec);
    }

    byte[]|mime:EncodeError bEnc = mime:base64EncodeBlob([1, 2, 3]);
    if bEnc is byte[] {
        byte[]|mime:DecodeError bDec = mime:base64DecodeBlob(bEnc);
        if bDec is byte[] {
            io:println(bDec.length());
        }
    }

    byte[] longInput = [];
    foreach int i in 0 ..< 100 {
        longInput.push(<byte>i);
    }
    byte[]|mime:EncodeError longEnc = mime:base64EncodeBlob(longInput);
    if longEnc is byte[] {
        string encStr = check string:fromBytes(longEnc);
        io:println(encStr.length() > 76);

        byte[]|mime:DecodeError longDec = mime:base64DecodeBlob(longEnc);
        if longDec is byte[] {
            io:println(longDec.length());
        }
    }

    string|byte[]|mime:DecodeError invalidDec = mime:base64Decode("not-valid-base64!!!");
    io:println(invalidDec is mime:DecodeError);
}
// @output SGVsbG8=
// @output Hello
// @output 3
// @output true
// @output 100
// @output true
