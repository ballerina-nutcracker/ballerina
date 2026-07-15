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

//go:build native_interp

// Package interpsrc — stub used when building the native interpreter binary.
// The native interpreter is always built from an already-extracted source tree,
// so there is no need to embed the source again.
package interpsrc

import "errors"

// ExtractTo is a no-op in native interpreter builds.
func ExtractTo(_, _ string) (string, error) {
	return "", errors.New("interpreter source is not embedded in native interpreter builds")
}
