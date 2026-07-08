// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
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

// Package stdlibs exposes Ballerina standard library packages baked into the
// interpreter binary. The embedded tree is laid out as <org>/<name>/<version>/
// <platform>/, matching FileSystemRepository's expectation. The all: prefix on
// the embed directive lets the build succeed against an empty tree (only a
// sentinel .gitkeep) until a real stdlib lands under ballerina/.
package stdlibs

import "embed"

//go:embed all:ballerina
var FS embed.FS
