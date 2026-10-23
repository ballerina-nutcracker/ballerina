# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/cache` package, an in-memory LRU cache with entry expiry and periodic cleanup.

## [cache](https://github.com/ballerina-platform/module-ballerina-cache/blob/master/docs/spec/spec.md)

An in-memory key-value cache that evicts the least recently used entries by an eviction factor once it is full, expires entries by a per-entry or default max-age, and optionally removes expired entries with a recurring `ballerina/task` job.

Types: `cache:Cache` (class), `cache:CacheConfig`, `cache:EvictionPolicy` (`LRU`), `cache:AbstractCache`, `cache:Error`.

| Function | Notes |
|---|---|
| `new cache:Cache(capacity?, evictionFactor?, evictionPolicy?, defaultMaxAge?, cleanupInterval?)` | An invalid configuration panics with jBallerina's `Validation failed for '...' constraint(s).` message, at run time rather than compile time |
| `put(key, value, maxAge?)` | Returns a `cache:Error` for a `()` value; when the cache is full, first evicts `ceil(capacity * evictionFactor)` least recently used entries |
| `get(key)` | Marks the entry most recently used; returns a `cache:Error` for a missing or expired key |
| `invalidate(key)` / `invalidateAll()` | `invalidate` returns a `cache:Error` for a missing key |
| `hasKey(key)` / `keys()` / `size()` / `capacity()` | Expired entries count until `get` or the cleanup task removes them; `keys` lists them from least to most recently used |
