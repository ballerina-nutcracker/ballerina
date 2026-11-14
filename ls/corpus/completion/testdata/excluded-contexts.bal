type Item record {| int value; |};
function consume(int value) {}
public function main() {
    Item item = {value: 1};
    int result = item.value;
    consume(1);
}
