// Sub-module for multiA — deliberately NOT listed in package.json's
// "export" array, exercising the non-exported-module import restriction.

public function hiddenApi() returns string {
    return "hidden";
}
