# Ballerina Cache Library

## Overview

The `ballerina/cache` library provides an in-memory, LRU-evicting key-value cache with per-entry and default expiry and an optional periodic cleanup task, plus an abstract cache object type for custom implementations. The Go Native Interpreter supports the whole runtime API; jBallerina's compile-time validation of literal cache configurations is not yet supported.

## Key Functionalities

- Create a cache with a capacity, eviction factor, eviction policy, default max-age and cleanup interval (`cache:Cache`, `cache:CacheConfig`), validated with the same error messages as jBallerina
- Add, read, check and discard entries (`put`, `get`, `hasKey`, `invalidate`, `invalidateAll`)
- Inspect the cache (`keys`, `size`, `capacity`)
- Evict the least recently used entries by the eviction factor once the cache is full
- Expire entries by a per-entry `maxAge` or the cache's `defaultMaxAge`, with a recurring `ballerina/task` job removing expired entries every `cleanupInterval` seconds
- Write custom caches against the `cache:AbstractCache` object type

## Examples

```ballerina
import ballerina/cache;
import ballerina/io;

public function main() returns error? {
    cache:Cache c = new (capacity = 4, evictionFactor = 0.5);
    check c.put("a", "A");
    check c.put("b", 2);
    check c.put("c", [1, 2]);
    _ = check c.get("a");
    check c.put("d", {x: 1});
    check c.put("e", "E"); // full: evicts the two least recently used entries, "b" and "c"
    io:println(c.size(), " ", c.hasKey("b"), " ", c.hasKey("a")); // 3 false true

    check c.put("session", "token", 30); // valid for 30 seconds
    any|cache:Error missing = c.get("zz");
    if missing is cache:Error {
        io:println(missing.message()); // Cache entry from the given key: zz, is not available.
    }
}
```

## Go Native Interpreter Support Status

This library is currently being migrated to Go to support the Ballerina Native Interpreter. The table below outlines the current support level for various features of this library in the Go implementation.

Support Levels:

- **Supported**: Fully implemented and tested in the Go version.
- **Partially Supported**: Implemented but lacking some edge cases, options, or sub-features. (See comments).
- **Not Yet Supported**: Planned for migration, but not yet implemented.
- **Cannot Support**: Cannot be implemented in the Go version due to technical limitations or architectural differences. (See comments).

| Feature/API | Support Status | Comments / Limitations |
|---|---|---|
| Module error type | Supported | `cache:Error` is a `distinct error` |
| Abstract cache object type | Supported | `cache:AbstractCache` |
| Cache configuration record | Supported | `cache:CacheConfig`; the fields carry no `@constraint` annotations because `ballerina/constraint` is not yet ported — `Cache.init` checks the same bounds itself |
| Eviction policy | Supported | `cache:EvictionPolicy` with `LRU` |
| Cache creation and configuration validation | Supported | An invalid configuration panics with the `constraint:validate` message jBallerina produces, e.g. `Validation failed for '$.capacity:minValue' constraint(s).` |
| Compile-time configuration validation | Not Yet Supported | jBallerina's compiler plugin (`CACHE_101`–`CACHE_105`) needs compiler plugin support in the interpreter; until then an invalid literal configuration compiles and `Cache.init` panics with the validation message at run time |
| Add an entry | Supported | `put`; a `()` value, or a `maxAge` that is neither `-1` nor positive, returns a `cache:Error` (see Notable Behavioural Changes); overwriting an existing key never evicts |
| Read an entry | Supported | `get`; marks the entry most recently used and removes it if it has expired |
| Discard an entry | Supported | `invalidate` |
| Discard all entries | Supported | `invalidateAll` |
| Check for a key | Supported | `hasKey`; an expired entry is removed and reported as absent (see Notable Behavioural Changes); recency is unchanged |
| List the keys | Supported | `keys`; the order differs from jBallerina (see Notable Behavioural Changes) |
| Cache size | Supported | `size`; expired entries are removed lazily, so it counts expired entries not yet removed by `get`, `hasKey`, `invalidate` or the cleanup task |
| Cache capacity | Supported | `capacity` |
| Least recently used eviction | Supported | When `put` finds the cache full it evicts `ceil(capacity * evictionFactor)` entries, computed exactly from the decimal value of `evictionFactor` and clamped to between 1 and `capacity` (see Notable Behavioural Changes) |
| Entry expiry | Supported | `maxAge` and `defaultMaxAge`, compared at nanosecond precision (see Notable Behavioural Changes) |
| Periodic cleanup | Supported | `cleanupInterval` schedules a recurring `ballerina/task` job that removes expired entries; as in jBallerina, a run that starts while the previous one is still scanning is skipped |

### Notable Behavioural Changes

- **Keys are listed from least to most recently used.** jBallerina returns `keys()` in the iteration order of a Java `ConcurrentHashMap`, which depends on key hashes and was never specified; the Go-native version returns them in recency order.
- **Capacities above the 32-bit range are honoured.** jBallerina narrows `capacity` to a Java `int`, so `capacity = 4294967297` behaves as a capacity of 1 during eviction while `capacity()` still returns 4294967297; the Go-native version uses the full `int` value.
- **The eviction count is computed exactly and clamped to between 1 and the capacity.** jBallerina evicts `ceil(capacity * evictionFactor)` entries computed in 32-bit float arithmetic, so the rounding error can evict one entry too many (e.g. `capacity = 50, evictionFactor = 0.3` evicts 16 instead of 15) ([ballerina-library#9406](https://github.com/ballerina-platform/ballerina-library/issues/9406)); the count is also unclamped, so a tiny `evictionFactor` (e.g. `1.0E-300`) evicts nothing and the cache settles at `capacity + 1` entries ([ballerina-library#9400](https://github.com/ballerina-platform/ballerina-library/issues/9400)), and for capacities above 2^24 the product can round up past `capacity` (e.g. `capacity = 16777219, evictionFactor = 1.0` gives 16777220) and fail the `put` with a `NullPointerException`. The Go-native version multiplies `capacity` by the shortest decimal representation of `evictionFactor` exactly and rounds up (so `capacity = 100, evictionFactor = 0.07` evicts 7 and `capacity = 10, evictionFactor = 0.25` evicts 3), then evicts at least one entry and at most every entry, so the cache never exceeds its capacity.
- **Overwriting a key in a full cache evicts nothing.** jBallerina checks the capacity before looking up the key, so `put` on an existing key in a full cache evicts `ceil(capacity * evictionFactor)` unrelated entries, possibly including the key being written ([ballerina-library#9401](https://github.com/ballerina-platform/ballerina-library/issues/9401)); the Go-native version replaces the value and marks the entry most recently used, since an overwrite does not grow the cache.
- **Expiry is compared at nanosecond precision.** jBallerina truncates both the expiry time and the current time to whole seconds, so an entry can expire up to one second early and a `maxAge` below 1 can expire it immediately ([ballerina-library#9399](https://github.com/ballerina-platform/ballerina-library/issues/9399)); the Go-native version compares Unix nanoseconds, and an expiry beyond the `int` range never expires.
- **`hasKey` and `invalidate` treat an expired entry as absent.** jBallerina's `hasKey` returns `true` for an expired entry that `get` then reports as missing, and `invalidate` succeeds on it ([ballerina-library#9402](https://github.com/ballerina-platform/ballerina-library/issues/9402)); the Go-native version removes the expired entry, so `hasKey` returns `false` and `invalidate` returns a `cache:Error`.
- **A `maxAge` that is neither -1 nor positive is rejected.** jBallerina's `put` silently replaces a `maxAge` of `0` or a negative value other than `-1` with `defaultMaxAge`, so with the default configuration `put(k, v, 0)` caches the entry forever ([ballerina-library#9404](https://github.com/ballerina-platform/ballerina-library/issues/9404)); the Go-native version returns a `cache:Error` such as `Unsupported max-age '0' for the key: k. It must be -1 or positive.`
