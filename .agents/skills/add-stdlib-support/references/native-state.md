# Associating native state with a Ballerina value

How a native extern recovers Go-side state (a parsed key, a compiled pattern, a connection handle) from a Ballerina value on a later call. The rule differs by value kind.

## Map/record values (`values.Map`) — use a package-private weak map

**Do not** add a field or accessor to `values.Map` for this. Ballerina mapping/record values are pure data with no encapsulation — a field added to `values.Map` becomes a general escape hatch every library could reach for, with no guarantee against another library overwriting or misusing the same slot, and no way to stop a user constructing a same-shaped value that never goes through your constructor. `runtime/values` must not carry any library-specific native-data association on `Map`.

Instead, keep the association entirely inside your own `native/` package as a package-private, GC-friendly weak map:

```go
var (
    stateMu sync.Mutex
    state   = make(map[weak.Pointer[values.Map]]any)
)

func setState(m *values.Map, data any) {
    wp := weak.Make(m)
    stateMu.Lock()
    state[wp] = data
    stateMu.Unlock()
    stdruntime.AddCleanup(m, func(p weak.Pointer[values.Map]) {
        stateMu.Lock()
        delete(state, p)
        stateMu.Unlock()
    }, wp)
}

func stateOf(m *values.Map) any {
    stateMu.Lock()
    defer stateMu.Unlock()
    return state[weak.Make(m)]
}
```

(`stdruntime` = Go's standard `"runtime"` package, aliased to avoid clashing with this project's `ballerina-lang-go/runtime`.) `weak.Pointer[T]` is documented as comparable and safe as a map key — two weak pointers compare equal iff their source pointers do — so pairing it with `runtime.AddCleanup` means the entry disappears once the value becomes unreachable: no leak, no runtime-level guarantee needed.

**Reference implementation:** `lib/stdlibs/ballerina/crypto/0.0.1/go1.2/native/keydata.go` (`setKeyData`/`keyDataOf`), which recovers the parsed `*rsa.PrivateKey`/`*ecdsa.PrivateKey`/etc. behind a `crypto:PrivateKey`/`PublicKey` record.

**Every reader must tolerate a miss** — nothing stops a user (or another library) from constructing a same-shaped value without ever calling your constructor. Return a clean domain error (e.g. crypto's `"Uninitialized ... key"`) rather than panicking or assuming the state is present.

If the association genuinely needs to be shared *across different modules/packages* (not just within your own native package), that's a rarer, separate case — flag it to the user explicitly rather than reaching for the pattern above; it would need a shared facility on `Env`/`Context` with clearly documented no-collision and cleanup-ownership caveats, which does not currently exist in this codebase.

## Object values (`values.Object`) — store the handle as an internal field

A Ballerina `object` (class instance) can only be constructed via `new` plus a class definition, so unlike maps/records, a user cannot fabricate a same-shaped object bypassing your constructor. `values.Object` has an established, sanctioned pattern: store the native handle directly as an internal field on the object, keyed by a name Ballerina code can't address.

**Reference implementation:** `os.go`'s `newProcessObject`, which stores a `pal.ProcessHandle` under the `"$handle"` field and reads it back via a small `getHandle(self *values.Object)` helper.

Keep using that pattern for objects; the weak-map approach above is specifically the map/record replacement for the now-removed `values.Map.SetNativeData`/`GetNativeData`.
