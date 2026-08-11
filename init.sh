#!/bin/bash

# Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
#
# WSO2 LLC. licenses this file to you under the Apache License,
# Version 2.0 (the "License"); you may not use this file except
# in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

# Build script for compiler-tools
# Builds all compiler-tools and places them in the root directory

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building compiler-tools..."

# Build tree-gen
echo "Building tree-gen..."
cd compiler-tools/tree-gen
go build -o ../../tree-gen
if [ $? -ne 0 ]; then
    echo "Error: Failed to build tree-gen"
    exit 1
fi
cd "$SCRIPT_DIR"
echo "✓ tree-gen built successfully"

echo ""
echo "All compiler-tools built successfully!"
echo "Executables are available in the root directory:"
echo "  - ./tree-gen"

# Generate test data files (corpus and parser/testdata JSON files)
echo ""
echo "Generating test data files..."
make update-testdata
echo "✓ Test data files generated"

