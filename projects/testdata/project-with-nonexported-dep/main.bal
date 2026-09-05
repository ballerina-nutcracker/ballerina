// Test project that imports a non-exported sub-module from a dependency.
// Reuses the existing mockorg/multiA testdata package (already exported:
// multiA, multiA.util) plus its multiA.hidden sub-module, which is
// deliberately not listed in multiA's exported modules.

import mockorg/multiA;
import mockorg/multiA.hidden;

public function main() {
    string _ = multiA:processValue();
    string _ = hidden:hiddenApi();
}
