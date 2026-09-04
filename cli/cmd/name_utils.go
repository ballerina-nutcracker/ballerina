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
	"os/user"
	"regexp"
	"strings"
)

// maxNameLength is the maximum allowed length for package and org names.
const maxNameLength = 256

// Validation regex patterns
var (
	// validPackageNamePattern matches alphanumerics, underscores, and dots.
	validPackageNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.]+$`)

	// allDotsPattern matches strings that contain only dots.
	allDotsPattern = regexp.MustCompile(`^\.+$`)

	// validOrgNamePattern matches alphanumerics and underscores only.
	validOrgNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	// startsWithDigitPattern matches strings starting with a digit.
	startsWithDigitPattern = regexp.MustCompile(`^[0-9]`)

	// consecutiveUnderscoresPattern matches two or more consecutive underscores.
	consecutiveUnderscoresPattern = regexp.MustCompile(`__+`)

	// invalidCharsPattern matches characters not allowed in package names.
	invalidCharsPattern = regexp.MustCompile(`[^a-zA-Z0-9_.]`)

	// invalidOrgCharsPattern matches characters not allowed in org names.
	invalidOrgCharsPattern = regexp.MustCompile(`[^a-zA-Z0-9_]`)

	// onlyNonAlphanumericPattern matches strings with no alphanumeric characters.
	onlyNonAlphanumericPattern = regexp.MustCompile(`^[^a-zA-Z0-9]+$`)
)

// validatePackageName validates a Ballerina package name.
// A valid package name:
//   - Contains only alphanumerics, underscores, and dots
//   - Is not all dots
//   - Does not start or end with underscore
//   - Does not contain consecutive underscores
//   - Does not start with a digit
//   - Is at most 256 characters
func validatePackageName(name string) bool {
	if name == "" {
		return false
	}

	if len(name) > maxNameLength {
		return false
	}

	// Must match valid characters pattern
	if !validPackageNamePattern.MatchString(name) {
		return false
	}

	// Cannot be all dots
	if allDotsPattern.MatchString(name) {
		return false
	}

	// Cannot start or end with underscore
	if strings.HasPrefix(name, "_") || strings.HasSuffix(name, "_") {
		return false
	}

	// Cannot have consecutive underscores
	if consecutiveUnderscoresPattern.MatchString(name) {
		return false
	}

	// Cannot start with digit
	if startsWithDigitPattern.MatchString(name) {
		return false
	}

	return true
}

// validateOrgName validates a Ballerina organization name.
// A valid org name contains only alphanumerics and underscores.
func validateOrgName(name string) bool {
	if name == "" {
		return false
	}

	if len(name) > maxNameLength {
		return false
	}

	return validOrgNamePattern.MatchString(name)
}

// guessPkgName attempts to derive a valid package name from the given name.
// If the name cannot be sanitized, returns "my_package".
func guessPkgName(packageName string) string {
	// If only non-alphanumeric characters, use default
	if packageName == "" || onlyNonAlphanumericPattern.MatchString(packageName) {
		return "my_package"
	}

	// Replace invalid characters with underscore
	result := invalidCharsPattern.ReplaceAllString(packageName, "_")

	// Prepend "app" if starts with digit
	if startsWithDigitPattern.MatchString(result) {
		result = "app" + result
	}

	// Remove leading underscores
	result = strings.TrimLeft(result, "_")

	// Replace consecutive underscores with single underscore
	result = consecutiveUnderscoresPattern.ReplaceAllString(result, "_")

	// Remove trailing underscores
	result = strings.TrimRight(result, "_")

	// If result is empty after sanitization, use default
	if result == "" {
		return "my_package"
	}

	return result
}

// guessOrgName attempts to derive a valid organization name from the system user.
// Falls back to "my_org" if the username cannot be determined or sanitized.
func guessOrgName() string {
	currentUser, err := user.Current()
	if err != nil || currentUser.Username == "" {
		return "my_org"
	}
	return deriveOrgNameFromUsername(currentUser.Username)
}

// deriveOrgNameFromUsername sanitizes an OS username into a valid org name,
// split out from guessOrgName so this logic can be tested directly against
// arbitrary input rather than only the real, environment-dependent OS user.
func deriveOrgNameFromUsername(username string) string {
	// On some systems, username may include domain (e.g., "DOMAIN\user")
	if idx := strings.LastIndex(username, "\\"); idx >= 0 {
		username = username[idx+1:]
	}

	// Validate the username as org name
	if validateOrgName(username) {
		return strings.ToLower(username)
	}

	// Sanitize: replace invalid characters with underscore
	sanitized := invalidOrgCharsPattern.ReplaceAllString(username, "_")

	// Remove leading/trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	// Replace consecutive underscores
	sanitized = consecutiveUnderscoresPattern.ReplaceAllString(sanitized, "_")

	// If empty after sanitization, use default
	if sanitized == "" {
		return "my_org"
	}

	return strings.ToLower(sanitized)
}
