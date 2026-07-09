# Adding a PAL method

**PAL constraint**: every platform interaction (io, http, fs, env, time) must go through the Platform Adaptation Layer, never the underlying Go stdlib directly. If the relevant PAL method doesn't exist, add it across three files:

1. **`platform/pal/platform.go`** — add new fields to the relevant struct (`IO`, `Time`, `FS`, `HTTP`, `OS`) or define a new struct if no existing category fits.

2. **`platform/palnative/`** — implement every new PAL field for the CLI build. Place FS methods in `fs.go`, OS methods in `os.go`, etc. If `test_util` needs to share the implementation, export `NewNative<Category>PAL()` so it can be called from there.

3. **`test_util/test_util.go` → `TestPal`** — wire new fields in. Safest pattern: start from `palnative.NewNative<Category>PAL()` and override only the test-specific fields.

**Silent-failure trap:** failing to update `TestPal` causes nil-pointer dereferences in corpus tests even when the CLI run succeeds.
