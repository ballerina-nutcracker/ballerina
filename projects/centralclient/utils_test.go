// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package centralclient

import (
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"

	"ballerina/common/bfs"
)

var tempBalaCache string

func TestMain(t *testing.M) {
	tempBalaCache = path.Join("build", "temp-test-utils-bala-cache")
	if err := os.MkdirAll(tempBalaCache, 0o755); err != nil {
		panic(err)
	}

	code := t.Run()

	if err := os.RemoveAll(tempBalaCache); err != nil {
		panic(err)
	}

	os.Exit(code)
}

func TestGetAsList(t *testing.T) {
	tests := []struct {
		name          string
		versionList   string
		expectedArray []string
	}{
		{
			name:          "empty array",
			versionList:   "[]",
			expectedArray: []string{},
		},
		{
			name:          "single version",
			versionList:   "[\"1.1.11\"]",
			expectedArray: []string{"1.1.11"},
		},
		{
			name:          "multiple versions",
			versionList:   "[\"1.0.0\", \"1.2.0\"]",
			expectedArray: []string{"1.0.0", "1.2.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			versions, err := getAsList(tt.versionList)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(versions) != len(tt.expectedArray) {
				t.Errorf("expected %d versions, got %d", len(tt.expectedArray), len(versions))
				return
			}
			for i, v := range versions {
				if v != tt.expectedArray[i] {
					t.Errorf("at index %d: expected %s, got %s", i, tt.expectedArray[i], v)
				}
			}
		})
	}
}

func TestValidatePackageVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		isValid bool
	}{
		{"valid simple version", "1.0.0", true},
		{"valid multi-digit version", "1.1.11", true},
		{"valid snapshot version", "2.2.2-snapshot", true},
		{"valid snapshot with number", "2.2.2-snapshot-1", true},
		{"valid alpha version", "2.2.2-alpha", true},
		{"invalid short version", "200", false},
		{"invalid four-part version", "2.2.2.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validatePackageVersion(tt.version, ClientContext{})

			if tt.isValid {
				if err != nil {
					t.Errorf("expected valid version, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error for invalid version: %s", tt.version)
				} else if !strings.Contains(err.Error(), "Invalid version:") {
					t.Errorf("error message should contain 'Invalid version:', got: %s", err.Error())
				}
			}
		})
	}
}

func TestJsonContentTypeChecker(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"plain application/json", "application/json", true},
		{"json with charset", "application/json; charset=utf-8", true},
		{"octet-stream", "application/octet-stream", false},
		{"text/plain", "text/plain", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isApplicationJSONContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("expected %v, got %v for content type: %s", tt.expected, result, tt.contentType)
			}
		})
	}
}

func TestGetBalaFileName(t *testing.T) {
	tests := []struct {
		name               string
		contentDisposition string
		balaFile           string
		expected           string
	}{
		{
			name:               "with content disposition",
			contentDisposition: "attachment; filename=org-package-any-1.0.0.bala",
			balaFile:           "fallback.bala",
			expected:           "org-package-any-1.0.0.bala",
		},
		{
			name:               "without content disposition",
			contentDisposition: "",
			balaFile:           "package-any-1.0.0.bala",
			expected:           "package-any-1.0.0.bala",
		},
		{
			name:               "invalid content disposition format",
			contentDisposition: "inline; name=test.bala",
			balaFile:           "fallback.bala",
			expected:           "fallback.bala",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBalaFileName(tt.contentDisposition, tt.balaFile)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetPlatformFromBala(t *testing.T) {
	tests := []struct {
		name        string
		balaName    string
		packageName string
		version     string
		expected    string
	}{
		{
			name:        "standard platform extraction",
			balaName:    "mypackage-any-1.0.0.bala",
			packageName: "mypackage",
			version:     "1.0.0",
			expected:    "any",
		},
		{
			name:        "java platform",
			balaName:    "http-java17-2.0.0.bala",
			packageName: "http",
			version:     "2.0.0",
			expected:    "java17",
		},
		{
			name:        "invalid format",
			balaName:    "invalid.bala",
			packageName: "mypackage",
			version:     "1.0.0",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPlatformFromBala(tt.balaName, tt.packageName, tt.version)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCreateBalaFileDirectory(t *testing.T) {
	fsys := bfs.NewMemFS()
	ctx := ClientContext{}

	tests := []struct {
		name      string
		dirPath   string
		expectErr bool
	}{
		{
			name:      "create single directory",
			dirPath:   "bala/org/pkg",
			expectErr: false,
		},
		{
			name:      "create nested directories",
			dirPath:   "bala/org/pkg/1.0.0/any",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := createBalaFileDirectory(fsys, tt.dirPath, ctx)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func TestCheckHashInternal(t *testing.T) {
	fsys := bfs.NewMemFS()

	tests := []struct {
		name     string
		filePath string
		content  []byte
		expected string
	}{
		{
			name:     "valid file hash",
			filePath: "test.txt",
			content:  []byte("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "empty file hash",
			filePath: "empty.txt",
			content:  []byte(""),
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "non-existent file",
			filePath: "nonexistent.txt",
			content:  nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.content != nil {
				_ = bfs.WriteFile(fsys, tt.filePath, tt.content, 0o644)
			}

			result, err := checkHashInternal(fsys, tt.filePath)
			if err != nil && tt.name != "non-existent file" {
				t.Errorf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHandleNightlyBuild(t *testing.T) {
	tests := []struct {
		name           string
		isNightlyBuild bool
		expectFile     bool
	}{
		{
			name:           "nightly build creates file",
			isNightlyBuild: true,
			expectFile:     true,
		},
		{
			name:           "non-nightly build does not create file",
			isNightlyBuild: false,
			expectFile:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := bfs.NewMemFS()
			ctx := ClientContext{}
			balaCachePath := "bala/org/pkg/1.0.0/any"

			err := handleNightlyBuild(tt.isNightlyBuild, fsys, balaCachePath, ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			nightlyFile := path.Join(balaCachePath, "nightly.build")
			_, statErr := fs.Stat(fsys, nightlyFile)

			if tt.expectFile && statErr != nil {
				t.Errorf("expected nightly.build file to exist, but got error: %v", statErr)
			}
			if !tt.expectFile && statErr == nil {
				t.Errorf("expected nightly.build file not to exist, but it does")
			}
		})
	}
}

func TestHandlePackageDeprecation(t *testing.T) {
	tests := []struct {
		name          string
		deprecateMsg  string
		expectFile    bool
		expectedError bool
	}{
		{
			name:          "with deprecation message",
			deprecateMsg:  "This package is deprecated",
			expectFile:    true,
			expectedError: false,
		},
		{
			name:          "without deprecation message",
			deprecateMsg:  "",
			expectFile:    false,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := bfs.NewMemFS()
			ctx := ClientContext{}
			balaCachePath := "bala/org/pkg/1.0.0/any"

			err := handlePackageDeprecation(tt.deprecateMsg, fsys, balaCachePath, ctx)
			if (err != nil) != tt.expectedError {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
			}

			deprecateFile := path.Join(balaCachePath, DeprecatedMetaFileName)
			data, statErr := fs.ReadFile(fsys, deprecateFile)

			if tt.expectFile {
				if statErr != nil {
					t.Errorf("expected deprecated.txt file to exist, but got error: %v", statErr)
				} else if string(data) != tt.deprecateMsg {
					t.Errorf("expected message %q, got %q", tt.deprecateMsg, string(data))
				}
			} else if !tt.expectFile && statErr == nil {
				t.Errorf("expected deprecated.txt file not to exist, but it does")
			}
		})
	}
}

func TestWriteDeprecatedMsg(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		fileExists  bool
		expectError bool
	}{
		{
			name:        "write message to existing file",
			message:     "Package deprecated",
			fileExists:  true,
			expectError: false,
		},
		{
			name:        "write message to non-existent file",
			message:     "Package deprecated",
			fileExists:  false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := bfs.NewMemFS()
			ctx := ClientContext{}
			metaFilePath := "bala/deprecated.txt"

			if tt.fileExists {
				_ = bfs.WriteFile(fsys, metaFilePath, []byte("old message"), 0o644)
			}

			err := writeDeprecatedMsg(fsys, metaFilePath, tt.message, ctx)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError && tt.fileExists {
				data, _ := fs.ReadFile(fsys, metaFilePath)
				if string(data) != tt.message {
					t.Errorf("expected message %q, got %q", tt.message, string(data))
				}
			}
		})
	}
}

func TestCreateMetaFile(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		errMsg      string
		expectError bool
	}{
		{
			name:        "create meta file successfully",
			filePath:    "bala/nightly.build",
			errMsg:      "failed to create meta file",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := bfs.NewMemFS()
			ctx := ClientContext{}

			err := createMetaFile(fsys, tt.filePath, tt.errMsg, ctx)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError {
				_, statErr := fs.Stat(fsys, tt.filePath)
				if statErr != nil {
					t.Errorf("expected file to exist, but got error: %v", statErr)
				}
			}
		})
	}
}
