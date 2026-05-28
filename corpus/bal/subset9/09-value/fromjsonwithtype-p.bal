type Person record {|
    string name;
    int age;
|};

public function main() {
    json missing = {"name": "Alice"};
    Person p = checkpanic missing.fromJsonWithType(Person); // @panic
}
