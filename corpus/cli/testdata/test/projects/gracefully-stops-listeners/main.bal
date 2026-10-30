import ballerina/http;
import ballerina/lang.runtime;
import ballerina/io;

function onStop() returns error? {
    io:println("GRACEFUL_STOP_HANDLER_RAN");
}

function init() {
    runtime:onGracefulStop(onStop);
}

service /echo on new http:Listener(20292) {
    resource function get hello() returns string {
        return "hi";
    }
}
