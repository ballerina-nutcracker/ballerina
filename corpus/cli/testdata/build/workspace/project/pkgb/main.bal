import ballerina/io;
import mockorg/leafpkg;

public function main() {
    io:println("hello from pkgb");
    io:println(leafpkg:doubleValue(4));
}
