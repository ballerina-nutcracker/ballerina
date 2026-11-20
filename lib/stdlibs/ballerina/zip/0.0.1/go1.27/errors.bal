// Copyright (c) 2026 WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
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

# Represents the properties belonging to an error about a single entry.
#
# + entryName - Name of the entry the error concerns
public type EntryErrorDetail record {
    string entryName;
};

# Represents any error returned by the `zip` module.
public type Error distinct error;

# Represents an error caused by a file that is not a valid ZIP archive.
public type InvalidArchiveError distinct Error;

# Represents an error caused by an entry that does not exist in the archive.
public type EntryNotFoundError distinct Error;

// The three errors about a single entry are `distinct (Error & error<EntryErrorDetail>)` in jBallerina; this
// interpreter cannot yet make an error intersection distinct (ballerina-nutcracker/ballerina#1427).

# Represents an error caused by an entry the module cannot read, because it is
# encrypted or uses an unsupported compression method.
public type UnsupportedEntryError distinct Error;

# Represents an error caused by an entry whose name would write outside the
# target directory.
public type UnsafePathError distinct Error;

# Represents an error caused by an entry that exceeds the configured
# extraction limits.
public type LimitExceededError distinct Error;

# Represents an error caused by reading from or writing to the file system.
public type FileSystemError distinct Error;
