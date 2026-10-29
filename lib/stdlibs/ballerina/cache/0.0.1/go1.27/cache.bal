// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

import ballerina/task;
import ballerina/time;

# Represents configurations for the `cache:Cache` object.
#
# + capacity - Maximum number of entries allowed in the cache
# + evictionFactor - The factor by which the entries will be evicted once the cache is full
# + evictionPolicy - The policy which is used to evict entries once the cache is full
# + defaultMaxAge - The max-age in seconds which all the cache entries are valid. '-1' means, the entries are
#                   valid forever. This will be overwritten by the `maxAge` property set when inserting item into
#                   the cache
# + cleanupInterval - Interval (in seconds) of the timer task, which will clean up the cache
public type CacheConfig record {|
    int capacity = 100;
    float evictionFactor = 0.25;
    EvictionPolicy evictionPolicy = LRU;
    decimal defaultMaxAge = -1;
    decimal cleanupInterval?;
|};

# Possible types of eviction policy that can be passed into the `EvictionPolicy`.
public enum EvictionPolicy {
    LRU
}

type CacheEntry record {|
    any data;
    decimal expTime;  // exp time since epoch. calculated based on the `maxAge` parameter when inserting to map
|};

class Cleanup {

    *task:Job;
    private Cache cache;

    public function execute() {
        time:Utc currentUtc = time:utcNow();
        externCleanUp(self.cache, <decimal>currentUtc[0] + currentUtc[1]);
    }

    public isolated function init(Cache cache) {
        self.cache = cache;
    }
}

# The `cache:Cache` object, which is used for all the cache-related operations. It is not recommended to insert `()`
# as the value of the cache since it doesn't make any sense to cache a nil.
public isolated class Cache {

    *AbstractCache;

    private final int maxCapacity;
    private final EvictionPolicy evictionPolicy;
    private final float evictionFactor;
    private final decimal defaultMaxAge;

    # Initializes new `cache:Cache` instance.
    # ```ballerina
    # cache:Cache cache = new(capacity = 10, evictionFactor = 0.2);
    # ```
    #
    # + cacheConfig - Configurations for the `cache:Cache` object
    public isolated function init(*CacheConfig cacheConfig) {
        string? validationError = validateConfig(cacheConfig);
        if validationError is string {
            panic prepareError(validationError);
        }
        self.maxCapacity = cacheConfig.capacity;
        self.evictionPolicy = cacheConfig.evictionPolicy;
        self.evictionFactor = cacheConfig.evictionFactor;
        self.defaultMaxAge = cacheConfig.defaultMaxAge;

        externInit(self);
        decimal? interval = cacheConfig?.cleanupInterval;
        if interval is decimal {
            time:Utc currentUtc = time:utcNow();
            time:Utc newTime = time:utcAddSeconds(currentUtc, interval);
            time:Civil time = time:utcToCivil(newTime);
            var result = task:scheduleJobRecurByFrequency(new Cleanup(self), interval, startTime = time);
            if result is task:Error {
                panic prepareError(string `Failed to schedule the cleanup task: ${result.message()}`);
            }
        }
    }

    # Adds the given key value pair to the cache. If the cache previously contained a value associated with the
    # provided key, the old value will be replaced by the newly-provided value.
    # ```ballerina
    # check cache.put("Hello", "Ballerina");
    # ```
    #
    # + key - Key of the value to be cached
    # + value - Value to be cached. Value should not be `()`
    # + maxAge - The time in seconds for which the cache entry is valid. If the value is '-1', the entry is
    #                     valid forever.
    # + return - `()` if successfully added to the cache or a `cache:Error` if a `()` value is inserted to the cache.
    public isolated function put(string key, any value, decimal maxAge = -1) returns Error? {
        if value is () {
            return prepareError("Unsupported cache value '()' for the key: " + key + ".");
        }

        CacheEntry entry = {
            data: value,
            expTime: expiryTime(time:utcNow(), maxAge > 0d ? maxAge : self.defaultMaxAge)
        };
        return externPut(self, key, entry);
    }

    # Returns the cached value associated with the provided key.
    # ```ballerina
    # any value = check cache.get(key);
    # ```
    #
    # + key - Key of the cached value, which should be retrieved
    # + return - The cached value associated with the provided key or a `cache:Error` if the provided cache key is not
    #            exisiting in the cache or any error occurred while retrieving the value from the cache.
    public isolated function get(string key) returns any|Error {
        time:Utc currentUtc = time:utcNow();
        CacheEntry? entry = externGet(self, key, <decimal>currentUtc[0] + currentUtc[1]);
        if entry is CacheEntry {
            return entry.data;
        } else {
            return prepareError("Cache entry from the given key: " + key + ", is not available.");
        }
    }

    # Discards a cached value from the cache.
    # ```ballerina
    # check cache.invalidate(key);
    # ```
    #
    # + key - Key of the cache value, which needs to be discarded from the cache
    # + return - `()` if successfully discarded the value or a `cache:Error` if the provided cache key is not present
    #            in the cache
    public isolated function invalidate(string key) returns Error? {
        if !self.hasKey(key) {
            return prepareError("Cache entry from the given key: " + key + ", is not available.");
        }
        externRemove(self, key);
    }

    # Discards all the cached values from the cache.
    # ```ballerina
    # check cache.invalidateAll();
    # ```
    #
    # + return - `()` if successfully discarded all the values from the cache or a `cache:Error` if any error
    #            occurred while discarding all the values from the cache.
    public isolated function invalidateAll() returns Error? {
        externRemoveAll(self);
    }

    # Checks whether the given key has an associated cached value.
    # ```ballerina
    # boolean result = cache.hasKey(key);
    # ```
    #
    # + key - The key to be checked in the cache
    # + return - `true` if a cached value is available for the provided key or `false` if there is no cached value
    #            associated for the given key
    public isolated function hasKey(string key) returns boolean {
        return externHasKey(self, key);
    }

    # Returns a list of all the keys from the cache.
    # ```ballerina
    # string[] keys = cache.keys();
    # ```
    #
    # + return - Array of all the keys from the cache
    public isolated function keys() returns string[] {
        return externKeys(self);
    }

    # Returns the size of the cache.
    # ```ballerina
    # int result = cache.size();
    # ```
    #
    # + return - The size of the cache
    public isolated function size() returns int {
        return externSize(self);
    }

    # Returns the capacity of the cache.
    # ```ballerina
    # int result = cache.capacity();
    # ```
    #
    # + return - The capacity of the cache
    public isolated function capacity() returns int {
        return self.maxCapacity;
    }
}

isolated function expiryTime(time:Utc now, decimal maxAge) returns decimal {
    if maxAge == -1d {
        return -1;
    }
    return <decimal>now[0] + now[1] + maxAge;
}

// Mirrors the `ballerina/constraint` annotations jBallerina declares on `CacheConfig`, including the
// `constraint:validate` error message: failed constraints sorted by path.
isolated function validateConfig(CacheConfig config) returns string? {
    string[] failed = [];
    if config.capacity < 1 {
        failed.push("$.capacity:minValue");
    }
    decimal? interval = config?.cleanupInterval;
    if interval is decimal && interval <= 0d {
        failed.push("$.cleanupInterval:minValueExclusive");
    }
    if config.defaultMaxAge < -1d {
        failed.push("$.defaultMaxAge:minValue");
    }
    float evictionFactor = config.evictionFactor;
    if !(evictionFactor <= 1.0) {
        failed.push("$.evictionFactor:maxValue");
    }
    if !(evictionFactor > 0.0) {
        failed.push("$.evictionFactor:minValueExclusive");
    }
    if failed.length() == 0 {
        return ();
    }
    string message = "Validation failed for ";
    foreach int i in 0 ..< failed.length() {
        if i > 0 {
            message += ",";
        }
        message += "'" + failed[i] + "'";
    }
    return message + " constraint(s).";
}

isolated function externInit(Cache cache) = external;

isolated function externRemoveAll(Cache cache) = external;

isolated function externHasKey(Cache cache, string key) returns boolean = external;

isolated function externKeys(Cache cache) returns string[] = external;

isolated function externSize(Cache cache) returns int = external;

isolated function externPut(Cache cache, string key, any value) = external;

isolated function externGet(Cache cache, string key, decimal currentTime) returns CacheEntry? = external;

isolated function externRemove(Cache cache, string key) = external;

isolated function externCleanUp(Cache cache, decimal currentTime) = external;
