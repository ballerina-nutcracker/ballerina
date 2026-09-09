import ballerina/io;
import ballerina/mime;

public function main() returns error? {
    mime:MediaType result = check mime:getMediaType("application/json; charset=UTF-8");
    io:println(result.primaryType);
    io:println(result.subType);
    io:println(result.suffix);
    io:println(result.getBaseType());
    io:println(result.parameters["charset"]);

    mime:MediaType result2 = check mime:getMediaType("application/svg+xml");
    io:println(result2.primaryType);
    io:println(result2.subType);
    io:println(result2.suffix);
    io:println(result2.getBaseType());

    mime:MediaType|error result3 = mime:getMediaType("invalid!!");
    if result3 is mime:MediaType {
        io:println("got media type: ", result3);
    }
    io:println("invalid content type");
}

// @output application
// @output json
// @output
// @output application/json
// @output UTF-8
// @output application
// @output svg
// @output xml
// @output application/svg
// @output invalid content type
