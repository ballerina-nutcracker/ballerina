import ballerina/http;

# A service representing a network-accessible API
# bound to port `9090`.
service / on new http:Listener(9090) {

    # A resource for generating greetings
    # + req - HTTP request containing the name
    # + return - string name with hello message or error
    resource function get greeting(http:Request req) returns string|error {
        string name = check req.getTextPayload();
        // Send a response back to the caller.
        if name.length() == 0 {
            return error("name should not be empty!");
        }
        return string `Hello, ${name}`;
    }
}
