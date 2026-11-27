# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/cache` package, an in-memory LRU cache with entry expiry and periodic cleanup.

## [cache](https://github.com/ballerina-platform/module-ballerina-cache/blob/master/docs/spec/spec.md)

An in-memory key-value cache that evicts the least recently used entries by an eviction factor once it is full, expires entries by a per-entry or default max-age, and optionally removes expired entries with a recurring `ballerina/task` job.

Types: `cache:Cache` (class), `cache:CacheConfig`, `cache:EvictionPolicy` (`LRU`), `cache:AbstractCache`, `cache:Error`.

| Function | Notes |
|---|---|
| `new cache:Cache(capacity?, evictionFactor?, evictionPolicy?, defaultMaxAge?, cleanupInterval?)` | An invalid configuration panics with jBallerina's `Validation failed for '...' constraint(s).` message, at run time rather than compile time |
| `put(key, value, maxAge?)` | Returns a `cache:Error` for a `()` value or a `maxAge` that is neither -1 nor positive; when a new key is added to a full cache, first evicts `ceil(capacity * evictionFactor)` least recently used entries (at least 1, at most `capacity`), while overwriting an existing key evicts nothing |
| `get(key)` | Marks the entry most recently used; returns a `cache:Error` for a missing or expired key |
| `invalidate(key)` / `invalidateAll()` | `invalidate` returns a `cache:Error` for a missing key |
| `hasKey(key)` / `keys()` / `size()` / `capacity()` | `hasKey` removes an expired entry and returns `false`; `keys` and `size` include an expired entry until `get`, `hasKey`, `invalidate`, `invalidateAll` or the cleanup task removes it, a `put` of the same key replaces it, or it is among the least recently used entries evicted when `put` adds a new key to a full cache; `keys` lists them from least to most recently used |
