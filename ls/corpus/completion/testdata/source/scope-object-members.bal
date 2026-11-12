int sharedName = 1;
int globalValue = 2;
function helper() {}
class ScopeBox {
    int sharedName = 3;
    int memberOnly = globalValue;

    function memberMethod() {}

    function examine(int argument) {
        int localValue = argument;
        int target = localValue;
    }
}

public function main() {

}
