// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License. You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an "AS IS"
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing permissions
// and limitations under the License.

package corpus

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/ls/server"
)

var update = flag.Bool("update", false, "update LSP corpus golden files")

type transcript struct {
	Source   string            `json:"source"`
	Sources  map[string]string `json:"sources"`
	Root     string            `json:"-"`
	Messages []json.RawMessage `json:"messages"`
	Expected []json.RawMessage `json:"expected"`
}

// The reader is replaced between batches; the server and output buffer are reused.
type stepTransport struct {
	reader io.Reader
	writer bytes.Buffer
}

func (t *stepTransport) Read(p []byte) (int, error)  { return t.reader.Read(p) }
func (t *stepTransport) Write(p []byte) (int, error) { return t.writer.Write(p) }

type segment struct {
	input     bytes.Buffer
	diskWrite *diskWriteAction
}
type diskWriteAction struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	remove  bool
}

const diskWriteMethod = "$test/writeFile"
const diskDeleteMethod = "$test/deleteFile"

func TestCorpus(t *testing.T) {
	t.Parallel()
	fixtures, err := discoverTranscripts(os.DirFS("."))
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			t.Parallel()
			runTranscript(t, fixture)
		})
	}
}

func discoverTranscripts(files fs.FS) ([]string, error) {
	paths, err := fs.Glob(files, "*/testdata/*.json")
	if err != nil {
		return nil, err
	}
	for _, name := range paths {
		if _, err := readTranscript(files, name); err != nil {
			return nil, err
		}
	}
	return paths, nil
}

func readTranscript(files fs.FS, name string) (transcript, error) {
	var fixture transcript
	content, err := fs.ReadFile(files, name)
	if err != nil {
		return fixture, err
	}
	if err := json.Unmarshal(content, &fixture); err != nil {
		return fixture, fmt.Errorf("%s: %w", name, err)
	}
	if fixture.Messages == nil || fixture.Expected == nil {
		return fixture, fmt.Errorf("%s: transcript requires messages and expected arrays", name)
	}
	return fixture, nil
}

func TestTranscriptTemporaryRoots(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, fixture, document string
		wantOnDisk              bool
	}{
		{"without sources", "sync/testdata/incremental_edit.initialize.json", "incremental_edit.bal", false},
		{"with sources", "sync/testdata/overlay-over-disk.json", "main.bal", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runTranscriptWithCheck(t, tc.fixture, func(_ *workspace.ProjectService, _ *compile.CompilationService, root string) {
				if !filepath.IsAbs(root) || !strings.Contains(root, " ") {
					t.Fatalf("fixture root must exercise a native absolute path with spaces: %q", root)
				}
				info, err := os.Stat(root)
				if err != nil || !info.IsDir() {
					t.Fatalf("fixture root is not a directory: %v", err)
				}
				_, err = os.Stat(filepath.Join(root, tc.document))
				if tc.wantOnDisk && err != nil {
					t.Fatalf("source file must remain on disk: %v", err)
				}
				if !tc.wantOnDisk && !os.IsNotExist(err) {
					t.Fatalf("overlay-only document must remain absent from disk: %v", err)
				}
			})
		})
	}
}

func runTranscript(t *testing.T, fixturePath string) {
	t.Helper()
	runTranscriptWithCheck(t, fixturePath, nil)
}

func runTranscriptWithCheck(t *testing.T, fixturePath string, check func(*workspace.ProjectService, *compile.CompilationService, string)) {
	t.Helper()
	fixture, err := readTranscript(os.DirFS("."), fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	rootPath := materializeSources(t, fixture.Sources)
	rootURI, err := workspace.FileURIFromPath(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	fixture.Root = rootURI.String()
	source := ""
	if fixture.Source != "" {
		data, err := fs.ReadFile(os.DirFS("."), path.Join(path.Dir(fixturePath), fixture.Source))
		if err != nil {
			t.Fatal(err)
		}
		source = string(data)
	}
	messages, err := fixtureMessages(fixture.Messages, source, fixture.Root)
	if err != nil {
		t.Fatal(err)
	}
	transport := &stepTransport{}
	bus := event.New()
	defer bus.Close()
	projectService := workspace.New(os.DirFS, bus)
	compiler := compile.New(projectService, bus, compile.WithDebounce(0))
	defer compiler.Shutdown()
	srv := server.New(transport, projectService, compiler, bus)
	defer srv.Flush()
	var actual []json.RawMessage
	for _, seg := range buildSegments(messages) {
		if seg.diskWrite != nil {
			applyDiskWrite(t, rootPath, seg.diskWrite)
			continue
		}
		transport.reader = bytes.NewReader(seg.input.Bytes())
		transport.writer.Reset()
		if err := srv.Serve(); err != nil {
			t.Fatalf("Serve: %v", err)
		}
		batch, err := readMessages(transport.writer.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		actual = append(actual, batch...)
	}
	if check != nil {
		check(projectService, compiler, rootPath)
	}
	actualNorm := normalizeRoot(actual, fixture.Root)
	if *update {
		fixture.Expected = actualNorm
		updated, err := json.MarshalIndent(fixture, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixturePath, append(updated, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	if !reflect.DeepEqual(normalizeMessages(actualNorm), normalizeMessages(fixture.Expected)) {
		t.Fatalf("transcript output mismatch\nactual: %s\nexpected: %s", formatMessages(actual), formatMessages(fixture.Expected))
	}
}

func normalizeRoot(messages []json.RawMessage, root string) []json.RawMessage {
	if root == "" {
		return messages
	}
	out := make([]json.RawMessage, len(messages))
	for i, m := range messages {
		out[i] = []byte(strings.ReplaceAll(string(m), root, "${root}"))
	}
	return out
}

func buildSegments(messages []protocol.Message) []segment {
	var segments []segment
	current := &segment{}
	flush := func() {
		if current.input.Len() > 0 {
			segments = append(segments, *current)
			current = &segment{}
		}
	}
	for _, message := range messages {
		if message.Method == diskWriteMethod || message.Method == diskDeleteMethod {
			var action diskWriteAction
			if err := json.Unmarshal(message.Params, &action); err != nil {
				panic("invalid disk action: " + err.Error())
			}
			action.remove = message.Method == diskDeleteMethod
			flush()
			segments = append(segments, segment{diskWrite: &action})
			continue
		}
		if err := protocol.WriteMessage(&current.input, message); err != nil {
			panic("frame message: " + err.Error())
		}
	}
	flush()
	return segments
}

// Test-only writes remain separate from the production source-reading capability.
func applyDiskWrite(t *testing.T, root string, action *diskWriteAction) {
	t.Helper()
	if root == "" || !fs.ValidPath(action.Path) || action.Path == "." {
		t.Fatalf("disk action needs a fixture root and relative file: %q", action.Path)
	}
	file := filepath.Join(root, filepath.FromSlash(action.Path))
	if action.remove {
		if err := os.Remove(file); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(action.Content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fixtureMessages(rawMessages []json.RawMessage, source, root string) ([]protocol.Message, error) {
	replacement, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	messages := make([]protocol.Message, 0, len(rawMessages))
	for _, raw := range rawMessages {
		s := strings.ReplaceAll(string(raw), `"${source}"`, string(replacement))
		s = strings.ReplaceAll(s, "${root}", root)
		var message protocol.Message
		if err := json.Unmarshal([]byte(s), &message); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func readMessages(output []byte) ([]json.RawMessage, error) {
	reader := bufio.NewReader(bytes.NewReader(output))
	var messages []json.RawMessage
	for {
		message, err := protocol.ReadMessage(reader)
		if err == io.EOF {
			return messages, nil
		}
		if err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(message)
		if err != nil {
			return nil, err
		}
		messages = append(messages, encoded)
	}
}

func materializeSources(t *testing.T, sources map[string]string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspace with spaces")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, content := range sources {
		applyDiskWrite(t, root, &diskWriteAction{Path: rel, Content: content})
	}
	return root
}

func normalizeMessages(messages []json.RawMessage) []any {
	normalized := make([]any, len(messages))
	for i, message := range messages {
		_ = json.Unmarshal(message, &normalized[i])
	}
	return normalized
}
func formatMessages(messages []json.RawMessage) string {
	formatted, _ := json.MarshalIndent(messages, "", "  ")
	return string(formatted)
}
