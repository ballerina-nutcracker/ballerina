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

package main

import (
	"strings"
	"testing"
)

func TestValidatePackageName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid names
		{"simple name", "mypackage", true},
		{"with underscore", "my_package", true},
		{"with dot", "my.package", true},
		{"with numbers", "package123", true},
		{"mixed", "my_package.v1", true},
		{"single char", "a", true},

		// Invalid: empty
		{"empty string", "", false},

		// Invalid: starts with digit
		{"starts with digit", "123package", false},
		{"starts with digit and underscore", "1_package", false},

		// Invalid: starts or ends with underscore
		{"starts with underscore", "_package", false},
		{"ends with underscore", "package_", false},
		{"only underscore", "_", false},

		// Invalid: consecutive underscores
		{"consecutive underscores", "my__package", false},
		{"triple underscore", "my___package", false},

		// Invalid: all dots
		{"single dot", ".", false},
		{"multiple dots", "...", false},

		// Invalid: special characters
		{"with hyphen", "my-package", false},
		{"with space", "my package", false},
		{"with at sign", "my@package", false},

		// Invalid: too long
		{"too long", strings.Repeat("a", 257), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := validatePackageName(tt.input)
			if result != tt.expected {
				t.Errorf("validatePackageName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateOrgName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid names
		{"simple name", "myorg", true},
		{"with underscore", "my_org", true},
		{"with numbers", "org123", true},
		{"uppercase", "MyOrg", true},
		{"single char", "a", true},

		// Invalid: empty
		{"empty string", "", false},

		// Invalid: contains dot (unlike package names)
		{"with dot", "my.org", false},

		// Invalid: special characters
		{"with hyphen", "my-org", false},
		{"with space", "my org", false},

		// Invalid: too long
		{"too long", strings.Repeat("a", 257), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := validateOrgName(tt.input)
			if result != tt.expected {
				t.Errorf("validateOrgName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGuessPkgName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Already valid
		{"valid name", "mypackage", "mypackage"},
		{"valid with underscore", "my_package", "my_package"},

		// Empty or non-alphanumeric
		{"empty string", "", "my_package"},
		{"only special chars", "---", "my_package"},
		{"only dots", "...", "my_package"},

		// Replace invalid chars
		{"with hyphen", "my-package", "my_package"},
		{"with space", "my package", "my_package"},
		{"with at sign", "my@package", "my_package"},

		// Starts with digit
		{"starts with digit", "123package", "app123package"},
		{"only digits", "123", "app123"},

		// Leading underscore (after replacement)
		{"leading hyphen", "-package", "package"},

		// Trailing underscore
		{"trailing hyphen", "package-", "package"},

		// Consecutive underscores
		{"multiple hyphens", "my--package", "my_package"},
		{"hyphen and underscore", "my-_package", "my_package"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := guessPkgName(tt.input)
			if result != tt.expected {
				t.Errorf("guessPkgName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGuessOrgName(t *testing.T) {
	t.Parallel()
	// guessOrgName depends on the current user, so we just verify it returns
	// a valid org name and is lowercase
	result := guessOrgName()

	if result == "" {
		t.Error("guessOrgName() returned empty string")
	}

	if !validateOrgName(result) {
		t.Errorf("guessOrgName() returned invalid org name: %q", result)
	}

	if result != strings.ToLower(result) {
		t.Errorf("guessOrgName() should return lowercase, got: %q", result)
	}
}

func TestDeriveOrgNameFromUsername(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		username string
		expected string
	}{
		{"already valid", "myorg", "myorg"},
		{"uppercase gets lowercased", "MyOrg", "myorg"},
		{"windows domain prefix stripped", `DOMAIN\myuser`, "myuser"},
		{"invalid chars sanitized", "my-org", "my_org"},
		{"consecutive underscores collapsed after sanitizing", "my--org", "my_org"},
		{"leading/trailing underscores trimmed after sanitizing", "-org-", "org"},
		{"empty after sanitizing falls back", "---", "my_org"},
		{"empty username falls back", "", "my_org"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := deriveOrgNameFromUsername(tt.username)
			if result != tt.expected {
				t.Errorf("deriveOrgNameFromUsername(%q) = %q, want %q", tt.username, result, tt.expected)
			}
		})
	}
}
