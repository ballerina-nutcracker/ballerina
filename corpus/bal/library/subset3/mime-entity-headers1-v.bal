import ballerina/io;
import ballerina/mime;

public function main() {
    mime:Entity entity = new ();
    entity.setHeader("Content-Type", "application/json");

    string|mime:HeaderNotFoundError ct = entity.getHeader("content-type");
    if ct is string {
        io:println(ct);
    }

    io:println(entity.hasHeader("content-type"));
    io:println(entity.hasHeader("accept"));

    entity.addHeader("Accept", "text/html");
    entity.addHeader("Accept", "application/json");
    string[]|mime:HeaderNotFoundError accepts = entity.getHeaders("accept");
    if accepts is string[] {
        io:println(accepts.length());
    }

    string[] names = entity.getHeaderNames();
    io:println(names.length());

    entity.removeHeader("Accept");
    io:println(entity.hasHeader("accept"));

    entity.removeAllHeaders();
    io:println(entity.hasHeader("content-type"));
}
// @output application/json
// @output true
// @output false
// @output 2
// @output 2
// @output false
// @output false
