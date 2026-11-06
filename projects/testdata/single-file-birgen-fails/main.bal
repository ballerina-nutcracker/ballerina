// Semantically valid, but BIR generation fails: xml[2] filler materialization
// is not yet implemented, so backend.GenBir fails without any earlier-phase
// diagnostics.
public function main() {
    xml[2] t = [];
    int n = t.length();
    _ = n;
}
