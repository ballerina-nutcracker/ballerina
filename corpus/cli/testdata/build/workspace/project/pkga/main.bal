import ballerina/io;
import mockorg/leafpkg;

public function main() {
    io:println("hello from pkga");
    io:println(leafpkg:getValue());
}
