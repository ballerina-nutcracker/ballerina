// Test project that imports from multi-module dependency

import mockorg/multiA;
import mockorg/multiA.util;

public function main() {
    string _ = multiA:processValue();
    int _ = util:doubleIt(42);
}
