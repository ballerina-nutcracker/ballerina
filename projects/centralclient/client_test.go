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
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"ballerina/common/bfs"

	"golang.org/x/tools/txtar"
)

const (
	testBalVersion = "slp5"
	testBalaName   = "sf-any.bala"
	accessToken    = "test-access-token"
)

var (
	utilTestResources = filepath.Join("testdata", "utils")
	update            = flag.Bool("update", false, "update expected test case outputs")
)

type TestRunner func(client CentralAPIClient) (string, string)

type TestCase struct {
	runner         TestRunner
	name           string
	filepath       string
	expectedOutput string
	expectedError  string
}

func TestCentralAPIClient(t *testing.T) {
	flag.Parse()

	testCases, err := parseTestCases("testdata")
	if err != nil {
		t.Fatalf("failed to parse test cases: %v", err)
	}

	client := newMockCentralAPIClient(t)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, errStr := tc.runner(client)

			if *update {
				if err := updateTestCase(tc, output, errStr); err != nil {
					t.Fatalf("failed to update test case %s: %v", tc.name, err)
				}
				t.Logf("blessed test case %s", tc.name)
				return
			}

			if output != tc.expectedOutput {
				t.Errorf("output mismatch:\nwant: %q\ngot:  %q", tc.expectedOutput, output)
			}
			if errStr != tc.expectedError {
				t.Errorf("error mismatch:\nwant: %q\ngot:  %q", tc.expectedError, errStr)
			}
		})
	}
}

func TestPullPackageSuccessWithDeprecation(t *testing.T) {
	balaContent, err := fs.ReadFile(os.DirFS(utilTestResources), testBalaName)
	if err != nil {
		t.Fatalf("failed to read bala file: %v", err)
	}

	expectedBalaFileName := "sf-2020r2-any-1.3.5.bala"
	mockClient := newDeprecatedPackageMockClient(balaContent, expectedBalaFileName)
	client := newTestCentralAPIClient(mockClient)

	memFS := bfs.NewMemFS()
	clientContext := ClientContext{IsBuild: false}

	err = client.PullPackage("wso2", "sf", "1.3.5", memFS,
		path.Join("bala", "wso2", "sf"), "2020r2-any", testBalVersion, clientContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	balaDir := path.Join("bala", "wso2", "sf", "1.3.5", "2020r2-any")
	deprecatedFile := path.Join(balaDir, "deprecated.txt")

	if _, err := fs.Stat(memFS, deprecatedFile); err != nil {
		t.Errorf("deprecated.txt file does not exist")
		return
	}

	content, err := fs.ReadFile(memFS, deprecatedFile)
	if err != nil {
		t.Errorf("failed to read deprecated.txt: %v", err)
		return
	}

	if !strings.Contains(string(content), "This package is deprecated") {
		t.Errorf("deprecated.txt does not contain expected message, got: %s", string(content))
	}
}

func TestPullPackageConnectionResetRetry(t *testing.T) {
	balaContent, err := os.ReadFile(filepath.Join(utilTestResources, testBalaName))
	if err != nil {
		t.Fatalf("failed to read bala file: %v", err)
	}

	expectedBalaFileName := "sf-2020r2-any-1.3.5.bala"
	attemptCount := 0
	downloadAttempts := 0

	mockClient := newRetryMockClient(balaContent, expectedBalaFileName, &attemptCount, &downloadAttempts)
	client := newTestCentralAPIClient(mockClient)

	memFS := bfs.NewMemFS()
	clientContext := ClientContext{IsBuild: false}

	err = client.PullPackage("foo", "sf", "1.3.5", memFS,
		path.Join("bala", "foo", "sf"), "2020r2-any", testBalVersion, clientContext)
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("expected 3 total attempts (2 failures + 1 success), got %d", attemptCount)
	}

	balaDir := path.Join("bala", "foo", "sf", "1.3.5", "2020r2-any")
	requiredFiles := []string{"bala.json", "package.json"}

	for _, file := range requiredFiles {
		filePath := path.Join(balaDir, file)
		if _, err := fs.Stat(memFS, filePath); err != nil {
			t.Errorf("required file does not exist after retry: %s", file)
		}
	}
}

func parseTestCases(dir string) ([]TestCase, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var testCases []TestCase
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".txtar") {
			continue
		}

		filepath := filepath.Join(dir, file.Name())
		archive, err := txtar.ParseFile(filepath)
		if err != nil {
			return nil, err
		}

		testCase, err := parseTestCase(archive, filepath)
		if err != nil {
			return nil, err
		}

		testCases = append(testCases, testCase)
	}

	return testCases, nil
}

func parseTestCase(archive *txtar.Archive, filepath string) (TestCase, error) {
	if archive == nil || len(archive.Files) < 3 {
		return TestCase{}, fmt.Errorf("invalid test case archive: expected at least 3 files")
	}

	runner := parseInput(archive.Files[0])

	return TestCase{
		runner:         runner,
		name:           strings.TrimSuffix(path.Base(filepath), ".txtar"),
		filepath:       filepath,
		expectedOutput: strings.TrimSpace(string(archive.Files[1].Data)),
		expectedError:  strings.TrimSpace(string(archive.Files[2].Data)),
	}, nil
}

func parseInput(data txtar.File) TestRunner {
	content := strings.ReplaceAll(string(data.Data), "\r\n", "\n")
	lines := strings.Split(strings.TrimSpace(content), "\n")

	if len(lines) == 0 {
		panic(fmt.Sprintf("empty input data in test case: %s", data.Name))
	}

	command := lines[0]
	args := lines[1:]

	switch command {
	case "GetPackageVersions":
		return createGetPackageVersionsRunner(args)
	case "GetPackage":
		return createGetPackageRunner(args)
	case "GetConnectors":
		return createGetConnectorsRunner(args)
	case "GetConnector":
		return createGetConnectorRunner(args)
	case "GetTriggers":
		return createGetTriggersRunner(args)
	case "GetTrigger":
		return createGetTriggerRunner(args)
	case "PullPackage":
		return createPullPackageRunner(args)
	default:
		panic(fmt.Sprintf("unsupported test case type: %s (file: %s)", command, data.Name))
	}
}

func createGetPackageVersionsRunner(args []string) TestRunner {
	return func(client CentralAPIClient) (string, string) {
		versions, err := client.GetPackageVersions(args[0], args[1], args[2], args[3])
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("%v", versions), ""
	}
}

func createGetPackageRunner(args []string) TestRunner {
	return func(client CentralAPIClient) (string, string) {
		pkg, err := client.GetPackage(args[0], args[1], args[2], args[3], args[4])
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("org=%s name=%s version=%s", pkg.Organization, pkg.Name, pkg.Version), ""
	}
}

func createGetConnectorsRunner(args []string) TestRunner {
	return func(client CentralAPIClient) (string, string) {
		params := map[string]string{"q": args[0]}
		connectors, err := client.GetConnectors(params, args[1], args[2])
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("%v", connectors != nil), ""
	}
}

func createGetConnectorRunner(args []string) TestRunner {
	return func(client CentralAPIClient) (string, string) {
		connector, err := client.GetConnector(args[0], args[1], args[2])
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("%v", connector), ""
	}
}

func createGetTriggersRunner(args []string) TestRunner {
	return func(client CentralAPIClient) (string, string) {
		params := map[string]string{"q": args[0]}
		triggers, err := client.GetTriggers(params, args[1], args[2])
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("%v", triggers), ""
	}
}

func createGetTriggerRunner(args []string) TestRunner {
	return func(client CentralAPIClient) (string, string) {
		trigger, err := client.GetTrigger(args[0], args[1], args[2])
		if err != nil {
			return "", err.Error()
		}
		return fmt.Sprintf("%v", trigger), ""
	}
}

func createPullPackageRunner(args []string) TestRunner {
	return func(client CentralAPIClient) (string, string) {
		memFS := bfs.NewMemFS()
		clientContext := ClientContext{IsBuild: false}
		err := client.PullPackage(args[0], args[1], args[2], memFS, args[3], args[4], args[5], clientContext)
		if err != nil {
			return "", err.Error()
		}
		return "", ""
	}
}

func updateTestCase(tc TestCase, actualOutput, actualError string) error {
	archive, err := txtar.ParseFile(tc.filepath)
	if err != nil {
		return err
	}

	if len(archive.Files) < 3 {
		return fmt.Errorf("invalid archive structure")
	}

	formatData := func(s string) []byte {
		if s == "" {
			return []byte("\n")
		}
		return fmt.Appendf(nil, "%s\n\n", s)
	}

	archive.Files[1].Data = formatData(actualOutput)
	archive.Files[2].Data = formatData(actualError)

	return os.WriteFile(tc.filepath, txtar.Format(archive), 0o644)
}

func newMockCentralAPIClient(t *testing.T) CentralAPIClient {
	t.Helper()

	packageJSON, err := os.ReadFile(filepath.Join(utilTestResources, "package.json"))
	if err != nil {
		t.Fatalf("failed to read package.json: %v", err)
	}

	packageSearchJSON, err := os.ReadFile(filepath.Join(utilTestResources, "packageSearch.json"))
	if err != nil {
		t.Fatalf("failed to read packageSearch.json: %v", err)
	}

	balaContent, err := os.ReadFile(filepath.Join(utilTestResources, testBalaName))
	if err != nil {
		t.Fatalf("failed to read bala file: %v", err)
	}

	transport := newMockTransport(packageJSON, packageSearchJSON, balaContent)
	mockClient := http.Client{
		Transport:     transport,
		CheckRedirect: preventRedirect,
	}

	return newTestCentralAPIClient(mockClient)
}

func newTestCentralAPIClient(httpClient http.Client) CentralAPIClient {
	return &centralAPIClientImpl{
		baseURL:     "https://localhost:9090/registry",
		httpClient:  httpClient,
		accessToken: accessToken,
		maxRetries:  2,
	}
}

func newDeprecatedPackageMockClient(balaContent []byte, balaFileName string) http.Client {
	transport := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.String(), "/registry/packages/") {
			return &http.Response{
				StatusCode: http.StatusFound,
				Status:     http.StatusText(http.StatusFound),
				Body:       io.NopCloser(strings.NewReader("")),
				Header: http.Header{
					"Location":            []string{"https://fileserver.dev-central.ballerina.io/2.0/wso2/sf/1.3.5/" + balaFileName},
					"Content-Disposition": []string{"attachment; filename=" + balaFileName},
					"Is-Deprecated":       []string{"true"},
					"Deprecate-Message":   []string{"This package is deprecated. Please use the new version."},
				},
				Request:    req,
				Proto:      "HTTP/1.1",
				ProtoMinor: 1,
				ProtoMajor: 1,
				Close:      true,
			}, nil
		}

		return newBinaryResponse(http.StatusOK, balaContent, req), nil
	})

	return http.Client{
		Transport:     transport,
		CheckRedirect: preventRedirect,
	}
}

func newRetryMockClient(balaContent []byte, balaFileName string, attemptCount, downloadAttempts *int) http.Client {
	transport := RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.String(), "/registry/packages/") {
			*attemptCount++
			return &http.Response{
				StatusCode: http.StatusFound,
				Status:     http.StatusText(http.StatusFound),
				Body:       io.NopCloser(strings.NewReader("")),
				Header: http.Header{
					"Location":            []string{"https://fileserver.dev-central.ballerina.io/2.0/foo/sf/1.3.5/" + balaFileName},
					"Content-Disposition": []string{"attachment; filename=" + balaFileName},
				},
				Request:    req,
				Proto:      "HTTP/1.1",
				ProtoMinor: 1,
				ProtoMajor: 1,
				Close:      true,
			}, nil
		}

		*downloadAttempts++
		if *downloadAttempts <= 2 {
			return nil, fmt.Errorf("Connection reset by peer")
		}

		resp := newBinaryResponse(http.StatusOK, balaContent, req)
		resp.Header.Set("RESOLVED_REQUESTED_URI", "https://fileserver.dev-central.ballerina.io/2.0/foo/sf/1.3.5/"+balaFileName)
		return resp, nil
	})

	return http.Client{
		Transport:     transport,
		CheckRedirect: preventRedirect,
	}
}

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func preventRedirect(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func newJSONResponse(status int, body string, req *http.Request) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Status:        http.StatusText(status),
		Body:          io.NopCloser(strings.NewReader(body)),
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		Request:       req,
		Proto:         "HTTP/1.1",
		ProtoMinor:    1,
		ProtoMajor:    1,
		ContentLength: int64(len(body)),
		Close:         true,
	}
}

func newBinaryResponse(status int, content []byte, req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Body:       io.NopCloser(strings.NewReader(string(content))),
		Header: http.Header{
			"Content-Type":   []string{ApplicationOctetStream},
			"Content-Length": []string{fmt.Sprintf("%d", len(content))},
		},
		Request:       req,
		Proto:         "HTTP/1.1",
		ProtoMinor:    1,
		ProtoMajor:    1,
		ContentLength: int64(len(content)),
		Close:         true,
	}
}

func newMockTransport(packageJSON, packageSearchJSON, balaContent []byte) http.RoundTripper {
	return RoundTripFunc(func(req *http.Request) (*http.Response, error) {
		path := req.URL.Path
		query := req.URL.Query()

		if strings.Contains(req.URL.Host, "fileserver.dev-central.ballerina.io") {
			if strings.HasSuffix(path, ".bala") {
				return newBinaryResponse(http.StatusOK, balaContent, req), nil
			}
		}

		switch path {
		// GetPackageVersions endpoints
		case "/registry/packages/wso2/sf":
			return newJSONResponse(http.StatusOK, `["1.0.0", "1.1.0", "1.2.0"]`, req), nil
		case "/registry/packages/unknown/package":
			return newJSONResponse(http.StatusNotFound, `{"message":"package not found: unknown/package:*_any"}`, req), nil
		case "/registry/packages/testorg/testpkg":
			return newJSONResponse(http.StatusUnauthorized, `{"message":"unauthorized access token for organization: 'testorg'"}`, req), nil
		case "/registry/packages/testorg/bad-pkg":
			return newJSONResponse(http.StatusBadRequest, `{"message":"invalid package name format"}`, req), nil
		case "/registry/packages/testorg/internalerror":
			return newJSONResponse(http.StatusInternalServerError, `{"message":"internal server error occurred"}`, req), nil
		case "/registry/packages/testorg/unavailable":
			return newJSONResponse(http.StatusServiceUnavailable, `{"message":"service temporarily unavailable"}`, req), nil
		case "/registry/packages/testorg/invalidjson":
			return newJSONResponse(http.StatusOK, `invalid json response`, req), nil

		// GetPackage endpoints
		case "/registry/packages/foo/winery/1.3.5":
			return newJSONResponse(http.StatusOK, string(packageJSON), req), nil
		case "/registry/packages/unknown/notfound/1.0.0":
			return newJSONResponse(http.StatusNotFound, `{"message":"package not found for: unknown/notfound:1.0.0"}`, req), nil
		case "/registry/packages/testorg/unauthorized/1.0.0":
			return newJSONResponse(http.StatusUnauthorized, `{"message":"unauthorized access token"}`, req), nil
		case "/registry/packages/testorg/badrequest/1.0.0":
			return newJSONResponse(http.StatusBadRequest, `{"message":"invalid version format"}`, req), nil
		case "/registry/packages/testorg/servererror/1.0.0":
			return newJSONResponse(http.StatusInternalServerError, `{"message":"database connection failed"}`, req), nil

		// GetConnectors endpoints
		case "/registry/connectors":
			return handleConnectorsRequest(query, packageSearchJSON, req)

		// GetConnector endpoints
		case "/registry/connectors/123":
			return newJSONResponse(http.StatusOK, `{"id": "123", "organization": "foo", "name": "winery", "version": "1.3.5"}`, req), nil
		case "/registry/connectors/notfound":
			return newJSONResponse(http.StatusNotFound, `{"message":"connector not found"}`, req), nil
		case "/registry/connectors/unauthorized":
			return newJSONResponse(http.StatusUnauthorized, `{"message":"unauthorized access"}`, req), nil
		case "/registry/connectors/invalidjson":
			return newJSONResponse(http.StatusOK, `invalid json`, req), nil
		case "/registry/connectors/servererror":
			return newJSONResponse(http.StatusInternalServerError, `{"message":"internal server error"}`, req), nil

		// GetTriggers endpoints
		case "/registry/triggers":
			return handleTriggersRequest(query, req)

		// GetTrigger endpoints
		case "/registry/triggers/456":
			return newJSONResponse(http.StatusOK, `{"id": "456", "name": "http-trigger", "type": "http"}`, req), nil
		case "/registry/triggers/notfound":
			return newJSONResponse(http.StatusNotFound, `{"message":"trigger not found"}`, req), nil
		case "/registry/triggers/unauthorized":
			return newJSONResponse(http.StatusUnauthorized, `{"message":"unauthorized access"}`, req), nil
		case "/registry/triggers/invalidjson":
			return newJSONResponse(http.StatusOK, `invalid json`, req), nil
		case "/registry/triggers/servererror":
			return newJSONResponse(http.StatusInternalServerError, `{"message":"internal server error"}`, req), nil

		default:
			if strings.HasPrefix(path, "/registry/packages/pullorg/") {
				return handlePullPackageRequest(path, balaContent, req)
			}
			return newJSONResponse(http.StatusNotFound, ``, req), nil
		}
	})
}

func handleConnectorsRequest(query map[string][]string, packageSearchJSON []byte, req *http.Request) (*http.Response, error) {
	var q string
	if values, ok := query["q"]; ok && len(values) > 0 {
		q = values[0]
	}

	switch q {
	case "unauthorized":
		return newJSONResponse(http.StatusUnauthorized, `{"message":"unauthorized access"}`, req), nil
	case "badrequest":
		return newJSONResponse(http.StatusBadRequest, `{"message":"invalid query parameter"}`, req), nil
	case "servererror":
		return newJSONResponse(http.StatusInternalServerError, `{"message":"internal server error"}`, req), nil
	default:
		return newJSONResponse(http.StatusOK, string(packageSearchJSON), req), nil
	}
}

func handleTriggersRequest(query map[string][]string, req *http.Request) (*http.Response, error) {
	var q string
	if values, ok := query["q"]; ok && len(values) > 0 {
		q = values[0]
	}

	switch q {
	case "unauthorized":
		return newJSONResponse(http.StatusUnauthorized, `{"message":"unauthorized access"}`, req), nil
	case "badrequest":
		return newJSONResponse(http.StatusBadRequest, `{"message":"invalid query parameter"}`, req), nil
	case "servererror":
		return newJSONResponse(http.StatusInternalServerError, `{"message":"internal server error"}`, req), nil
	default:
		return newJSONResponse(http.StatusOK, `{"count": 2, "triggers": [{"id": "1", "name": "trigger1"}]}`, req), nil
	}
}

func handlePullPackageRequest(path string, balaContent []byte, req *http.Request) (*http.Response, error) {
	switch path {
	case "/registry/packages/pullorg/pkg/1.0.0":
		return &http.Response{
			StatusCode: http.StatusFound,
			Status:     http.StatusText(http.StatusFound),
			Body:       io.NopCloser(strings.NewReader("")),
			Header: http.Header{
				"Location":            []string{"https://fileserver.dev-central.ballerina.io/2.0/pullorg/pkg/1.0.0/pkg-2020r2-any-1.0.0.bala"},
				"Content-Disposition": []string{"attachment; filename=pkg-2020r2-any-1.0.0.bala"},
			},
			Request:    req,
			Proto:      "HTTP/1.1",
			ProtoMinor: 1,
			ProtoMajor: 1,
			Close:      true,
		}, nil
	case "/2.0/pullorg/pkg/1.0.0/pkg-2020r2-any-1.0.0.bala":
		return newBinaryResponse(http.StatusOK, balaContent, req), nil
	default:
		if strings.Contains(path, "/notfound/") {
			return newJSONResponse(http.StatusNotFound, `{"message":"package not found: pullorg/notfound:1.0.0"}`, req), nil
		}
		if strings.Contains(path, "/unauthorized/") {
			return newJSONResponse(http.StatusUnauthorized, `{"message":"unauthorized access token for organization: 'pullorg'"}`, req), nil
		}
		if strings.Contains(path, "/badrequest/") {
			return newJSONResponse(http.StatusBadRequest, `{"message":"invalid package version format"}`, req), nil
		}
		if strings.Contains(path, "/servererror/") {
			return newJSONResponse(http.StatusInternalServerError, `{"message":"internal server error occurred"}`, req), nil
		}
		if strings.Contains(path, "/unavailable/") {
			return newJSONResponse(http.StatusServiceUnavailable, `{"message":"service temporarily unavailable"}`, req), nil
		}
		if strings.Contains(path, "/missing-location/") {
			return &http.Response{
				StatusCode: http.StatusFound,
				Status:     http.StatusText(http.StatusFound),
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{"Content-Disposition": []string{"attachment; filename=pkg-any-1.0.0.bala"}},
				Request:    req,
				Proto:      "HTTP/1.1",
				ProtoMinor: 1,
				ProtoMajor: 1,
				Close:      true,
			}, nil
		}
		if strings.Contains(path, "/missing-content-disposition/") {
			return &http.Response{
				StatusCode: http.StatusFound,
				Status:     http.StatusText(http.StatusFound),
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{"Location": []string{"https://fileserver.dev-central.ballerina.io/2.0/pullorg/pkg/1.0.0/pkg-any-1.0.0.bala"}},
				Request:    req,
				Proto:      "HTTP/1.1",
				ProtoMinor: 1,
				ProtoMajor: 1,
				Close:      true,
			}, nil
		}

		return newJSONResponse(http.StatusNotFound, ``, req), nil
	}
}
