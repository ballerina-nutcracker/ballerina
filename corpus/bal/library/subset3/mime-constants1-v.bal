import ballerina/io;
import ballerina/mime;

public function main() {
    io:println(mime:APPLICATION_JSON);
    io:println(mime:TEXT_PLAIN);
    io:println(mime:APPLICATION_OCTET_STREAM);
    io:println(mime:CONTENT_TYPE);
    io:println(mime:CONTENT_LENGTH);
    io:println(mime:DEFAULT_CHARSET);
    io:println(mime:MULTIPART_FORM_DATA);
    io:println(mime:IMAGE_JPEG);
}
// @output application/json
// @output text/plain
// @output application/octet-stream
// @output content-type
// @output content-length
// @output UTF-8
// @output multipart/form-data
// @output image/jpeg
