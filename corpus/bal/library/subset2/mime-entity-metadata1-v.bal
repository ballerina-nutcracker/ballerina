import ballerina/io;
import ballerina/mime;

public function main() returns error? {
    mime:Entity e = new ();
    mime:InvalidContentTypeError? err = e.setContentType("text/html");
    if !(err is mime:InvalidContentTypeError) {
        io:println(e.getContentType());
    }

    e.setContentId("cid-001");
    io:println(e.getContentId());

    e.setContentLength(512);
    int|error clen = e.getContentLength();
    if clen is int {
        io:println(clen);
    }
}
// @output text/html
// @output cid-001
// @output 512
