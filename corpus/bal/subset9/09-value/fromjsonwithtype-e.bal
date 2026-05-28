import ballerina/io;

public function main() {
    json arr = [1, 2];
    arr.fromJsonWithType(); // @error

    int x = 1;
    x.fromJsonWithType(int); // @error
}
