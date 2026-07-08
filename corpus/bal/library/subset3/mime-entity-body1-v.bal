import ballerina/io;
import ballerina/mime;

public function main() {
    mime:Entity textEntity = new ();
    textEntity.setText("Hello World");
    string|mime:ParserError textResult = textEntity.getText();
    if textResult is string {
        io:println(textResult);
    }

    mime:Entity bytesEntity = new ();
    bytesEntity.setByteArray([72, 101, 108, 108, 111]);
    byte[]|mime:ParserError bytesResult = bytesEntity.getByteArray();
    if bytesResult is byte[] {
        io:println(bytesResult.length());
    }

    mime:Entity bodyEntity = new ();
    bodyEntity.setBody("dispatched text");
    string|mime:ParserError dispatchResult = bodyEntity.getText();
    if dispatchResult is string {
        io:println(dispatchResult);
    }

    mime:Entity wrongEntity = new ();
    wrongEntity.setText("text");
    byte[]|mime:ParserError wrongResult = wrongEntity.getByteArray();
    if wrongResult is mime:ParserError {
        io:println("parser error");
    }
}
// @output Hello World
// @output 5
// @output dispatched text
// @output parser error
