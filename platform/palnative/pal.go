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

// Package palnative provides the native-CLI implementation of pal.Platform.
// The HTTP factory and its TLS plumbing live in http.go; IO is small enough
// to inline here. Other environments (e.g. WASM/web-editor) supply their own
// pal.Platform without importing this package.
package palnative

import (
	"os"
	"time"

	"ballerina-lang-go/platform/pal"
)

var processStart = time.Now()

// NewPlatform returns the native-CLI pal.Platform, wiring os.Stdout/Stderr for
// IO and NewHTTPClient for HTTP. The returned cleanup function releases signal
// resources owned by the platform.
func NewPlatform() (pal.Platform, func()) {
	signals, cleanupSignals := newSignalSource()
	return pal.Platform{
		IO: pal.IO{
			Stdout: func(p []byte) (n int, err error) { return os.Stdout.Write(p) },
			Stderr: func(p []byte) (n int, err error) { return os.Stderr.Write(p) },
		},
		FS: pal.FS{
			ReadFile: func(path string) ([]byte, error) {
				return os.ReadFile(path)
			},
			WriteFile: func(path string, data []byte) error {
				return os.WriteFile(path, data, 0o644)
			},
			AppendFile: func(path string, data []byte) (err error) {
				f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
				if err != nil {
					return err
				}
				defer func() {
					if cerr := f.Close(); cerr != nil && err == nil {
						err = cerr
					}
				}()
				_, err = f.Write(data)
				return err
			},
		},
		Time: pal.Time{
			Now:          time.Now,
			MonotonicNow: func() time.Duration { return time.Since(processStart) },
		},
		HTTP: pal.HTTP{
			NewClient: NewHTTPClient,
		},
		Signals: signals,
	}, cleanupSignals
}
