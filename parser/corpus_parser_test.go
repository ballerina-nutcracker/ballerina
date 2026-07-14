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

package parser

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ballerina-lang-go/parser/tree"
	"ballerina-lang-go/test_util"
	"ballerina-lang-go/tools/text"

	"github.com/sergi/go-diff/diffmatchpatch"
)

var update = flag.Bool("update", false, "update expected JSON files")

// Tests skipped because the parser stage cannot process them today.
var corpusParserSkipList = []string{
	// No skipped tests
}

// Regex parser ignore list (13 files: regex parser not yet implemented)
var regexParserIgnoreList = []string{
	"bala/test_bala/types/regexp_type_test.bal",
	"bala/test_projects/test_project_regexp/regexpTypes.bal",
	"jvm/largeMethods/modules/functions/large-functions.bal",
	"query/query-action.bal",
	"query/query-expr-with-query-construct-type.bal",
	"query/query_action_or_expr.bal",
	"query/simple-query-with-defined-type.bal",
	"types/readonly/test_inherently_immutable_type.bal",
	"types/readonly/test_selectively_immutable_type.bal",
	"types/regexp/regexp_type_test.bal",
	"types/regexp/regexp_value_negative_test.bal",
	"types/regexp/regexp_value_test.bal",
	"types/table/table_key_field_value_test.bal",
}

// shouldIgnoreFile checks if a file should be ignored based on the ignore lists
func shouldIgnoreFile(filePath string) bool {
	allIgnoreLists := [][]string{
		regexParserIgnoreList,
	}

	for _, ignoreList := range allIgnoreLists {
		for _, ignorePath := range ignoreList {
			if strings.HasSuffix(filePath, ignorePath) {
				return true
			}
		}
	}

	return false
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func TestParseCorpusFiles(t *testing.T) {
	if os.Getenv("GOARCH") == "wasm" {
		t.Skip("skipping parser testing wasm")
	}

	// Parser can parse all .bal files, not just -v.bal
	testPairs := test_util.GetTests(t, test_util.Parser, func(path string) bool {
		return true
	})

	// Create subtests for each file
	// Running in parallel for faster test execution
	for _, testPair := range testPairs {
		t.Run(testPair.Name, func(t *testing.T) {
			t.Parallel() // Run in parallel for faster execution (native only)
			slashed := filepath.ToSlash(testPair.InputPath)
			for _, skip := range corpusParserSkipList {
				if strings.HasSuffix(slashed, skip) {
					t.Skipf("Skipping parser test for %s", testPair.InputPath)
					return
				}
			}
			parseFile(t, testPair)
		})
	}
}

func TestJBalUnitTests(t *testing.T) {
	corpusDir := "./testdata/bal"
	if os.Getenv("GOARCH") == "wasm" {
		t.Skip("skipping parser testing wasm")
	}
	balFiles := getCorpusFiles(t, corpusDir)

	// Create subtests for each file
	// Running in parallel for faster test execution
	for _, balFile := range balFiles {
		// Skip files in ignore lists
		if shouldIgnoreFile(balFile) {
			t.Run(balFile, func(t *testing.T) {
				t.Skipf("Skipping file in ignore list: %s", balFile)
			})
			continue
		}

		// Create TestCase from file path
		testCase := createTestCase(t, balFile, corpusDir)

		t.Run(balFile, func(t *testing.T) {
			t.Parallel() // Run in parallel for faster execution (native only)
			parseFile(t, testCase)
		})
	}
}

func getCorpusFiles(t *testing.T, corpusBalDir string) []string {
	var balFiles []string
	err := filepath.Walk(corpusBalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".bal") {
			balFiles = append(balFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Error walking corpus/bal directory: %v", err)
	}

	if len(balFiles) == 0 {
		t.Fatalf("No .bal files found in %s", corpusBalDir)
	}
	return balFiles
}

func parseFile(t *testing.T, testCase test_util.TestCase) {
	// Catch any panics and convert them to errors
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()

	// Read file content
	content, readErr := os.ReadFile(testCase.InputPath)
	if readErr != nil {
		t.Fatalf("error reading file: %v", readErr)
	}

	reader := text.CharReaderFromText(string(content))

	lexer := NewLexer(reader)

	tokenReader := CreateTokenReader(lexer)

	ballerinaParser := NewBallerinaParserFromTokenReader(tokenReader)

	ast := ballerinaParser.Parse()

	actualJSON := tree.GenerateJSON(ast)

	normalizedJSON := normalizeJSON(actualJSON)

	// If update flag is set, check if update is needed and update if necessary
	if *update {
		if test_util.UpdateIfNeeded(t, testCase.ExpectedPath, normalizedJSON, normalizeJSON) {
			t.Fatalf("Updated expected JSON file: %s", testCase.ExpectedPath)
		}
		return
	}

	expectedJSON := expectedJSON(t, testCase.ExpectedPath)

	// Compare JSON strings exactly (no tolerance for formatting differences)
	if normalizedJSON != expectedJSON {
		diff := getDiff(expectedJSON, normalizedJSON)
		t.Errorf("JSON mismatch for %s\nExpected file: %s\n%s", testCase.InputPath, testCase.ExpectedPath, diff)
		return

	}
}

// createTestCase creates a TestCase from a file path and base directory
func createTestCase(t *testing.T, filePath string, baseDir string) test_util.TestCase {
	// Get the relative path from baseDir
	relPath, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		t.Fatalf("Failed to get relative path: %v", err)
	}

	// Replace "bal" directory with "parser" in the base directory path
	parserBaseDir := strings.Replace(baseDir, string(filepath.Separator)+"bal", string(filepath.Separator)+"parser", 1)

	// Construct the expected JSON path
	expectedJSONPath := filepath.Join(parserBaseDir, relPath)
	expectedJSONPath = strings.TrimSuffix(expectedJSONPath, ".bal") + ".json"

	return test_util.TestCase{
		Name:         filePath,
		InputPath:    filePath,
		ExpectedPath: expectedJSONPath,
	}
}

func normalizeJSON(jsonStr string) string {
	var obj any
	normalizedJSON := jsonStr
	if err := json.Unmarshal([]byte(jsonStr), &obj); err == nil {
		if normalized, err := json.MarshalIndent(obj, "", "  "); err == nil {
			normalizedJSON = string(normalized)
		}
	}
	return normalizedJSON
}

func expectedJSON(t *testing.T, expectedJSONPath string) string {
	// Check if file exists
	expectedJSONBytes, readErr := os.ReadFile(expectedJSONPath)
	if readErr != nil {
		t.Fatalf("error reading expected JSON file: %v", readErr)
		return ""
	}

	// File exists - normalize and compare
	expectedJSON := string(expectedJSONBytes)
	var expectedObj any
	if err := json.Unmarshal([]byte(expectedJSON), &expectedObj); err == nil {
		if normalized, err := json.MarshalIndent(expectedObj, "", "  "); err == nil {
			expectedJSON = string(normalized)
		}
	}
	return expectedJSON
}

// getDiff generates a detailed diff string showing differences between expected and actual AST strings.
func getDiff(expectedAST, actualAST string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(expectedAST, actualAST, false)
	return dmp.DiffPrettyText(diffs)
}
