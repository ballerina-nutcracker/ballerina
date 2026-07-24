import ballerina/io;
import mockorg/leafpkg;

public function main() {
    io:println("value: ", leafpkg:getValue());
}
