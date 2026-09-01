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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

package corpus

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/ls/server"
	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

var update = flag.Bool("update", false, "update LSP corpus golden files")

type transcript struct {
	Source   string            `json:"source"`  // single-file fixture: filename under testdata
	Sources  map[string]string `json:"sources"` // multi-file: relative path → content
	Root     string            `json:"-"`       // temp root path (set at runtime, not serialized)
	Messages []json.RawMessage `json:"messages"`
	Expected []json.RawMessage `json:"expected"`
}

// stepTransport feeds the server one segment of framed LSP messages at a
// time. The driver swaps the reader between server.Serve() calls so a
// $pal/writeFile action can run between segments (interleaved with the LSP
// notification sequence). Server state (initialized flag, the project service)
// persists across Serve() calls; only the input stream is reset.
type stepTransport struct {
	reader io.Reader
	writer bytes.Buffer
}

func (t *stepTransport) Read(p []byte) (int, error)  { return t.reader.Read(p) }
func (t *stepTransport) Write(p []byte) (int, error) { return t.writer.Write(p) }

// segment is one element of the transcript sequence: either a batch of framed
// LSP messages to feed the server, or a PAL-only file write the driver performs
// between LSP batches. A $pal/writeFile entry in the transcript becomes a
// palWrite segment.
type segment struct {
	input    bytes.Buffer // framed LSP messages for this batch
	palWrite *palWriteAction
}

type palWriteAction struct {
	Path    string `json:"path"` // relative to the fixture root
	Content string `json:"content"`
}

// $pal/writeFile is the sentinel method marking a driver-only file write. It
// is never sent to the server; the driver materializes the file through PAL.
const palWriteMethod = "$pal/writeFile"

func TestCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "sync/testdata/incremental_edit.initialize.json")
}

func TestShutdownExitCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "sync/testdata/shutdown-exit.json")
}

func TestURISchemeRejectionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "sync/testdata/uri-scheme-rejection.json")
}

func TestBuildProjectCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "sync/testdata/build-project.initialize.json")
}

func TestOverlayOverDiskCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "sync/testdata/overlay-over-disk.json")
}

func TestWatchedFileTransitionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "sync/testdata/watched-file-transition.json")
}

func TestStaleResultSuppressionCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "sync/testdata/stale-result-suppression.json")
}

func TestUnchangedRecompileSkipCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "sync/testdata/unchanged-recompile-skip.json")
}

func TestModifierChainIdenticalOutputCorpus(t *testing.T) {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()
	runTranscript(t, platform, "sync/testdata/modifier-chain-identical-output.json")
}

func runTranscript(t *testing.T, platform pal.Platform, fixturePath string) {
	t.Helper()
	content, err := platform.FS.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture transcript
	if err := json.Unmarshal(content, &fixture); err != nil {
		t.Fatal(err)
	}
	rootPath := ""
	if len(fixture.Sources) > 0 {
		rootPath = materializeSources(t, platform, fixture.Sources)
		fixture.Root = rootPath
	}
	source := ""
	if fixture.Source != "" {
		bytes, err := platform.FS.ReadFile(path.Join(path.Dir(fixturePath), fixture.Source))
		if err != nil {
			t.Fatal(err)
		}
		source = string(bytes)
	}
	messages, err := fixtureMessages(fixture.Messages, source, rootPath)
	if err != nil {
		t.Fatal(err)
	}
	segments := buildSegments(messages)
	transport := &stepTransport{}
	bus := event.New()
	defer bus.Close()
	projectService := workspace.New(platform, bus)
	compiler := compile.New(projectService, bus, compile.WithDebounce(0))
	defer compiler.Shutdown()
	srv := server.New(transport, projectService, compiler, bus)
	defer srv.Flush()
	var actual []json.RawMessage
	for _, seg := range segments {
		if seg.palWrite != nil {
			applyPalWrite(t, platform, rootPath, seg.palWrite)
			continue
		}
		transport.reader = bytes.NewReader(seg.input.Bytes())
		transport.writer.Reset()
		if err := srv.Serve(); err != nil {
			t.Fatalf("Serve: %v", err)
		}
		segActual, err := readMessages(transport.writer.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		actual = append(actual, segActual...)
	}
	if *update {
		fixture.Expected = actual
		// Normalize the root path back to ${root} placeholder in the golden.
		if fixture.Root != "" {
			for i := range fixture.Expected {
				s := string(fixture.Expected[i])
				s = strings.ReplaceAll(s, fixture.Root, "${root}")
				fixture.Expected[i] = []byte(s)
			}
		}
		updated, err := json.MarshalIndent(fixture, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := platform.FS.WriteFile(fixturePath, append(updated, '\n')); err != nil {
			t.Fatal(err)
		}
		return
	}
	// Normalize root path before comparison so the golden's ${root} placeholder
	// matches the actual temp path.
	actualNorm := actual
	expectedNorm := fixture.Expected
	if fixture.Root != "" {
		actualNorm = make([]json.RawMessage, len(actual))
		for i, m := range actual {
			s := string(m)
			s = strings.ReplaceAll(s, fixture.Root, "${root}")
			actualNorm[i] = []byte(s)
		}
	}
	if !reflect.DeepEqual(normalizeMessages(actualNorm), normalizeMessages(expectedNorm)) {
		t.Fatalf("transcript output mismatch\nactual: %s\nexpected: %s", formatMessages(actual), formatMessages(fixture.Expected))
	}
}

// buildSegments splits the substituted LSP message stream into server-fed
// batches separated by $pal/writeFile driver actions. A sentinel message is
// decoded into a palWrite segment; consecutive LSP messages are framed into one
// batch. Fixtures without sentinels yield a single batch — identical to the
// former single-Serve driver — so existing goldens stay byte-identical.
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
		if message.Method == palWriteMethod {
			var action palWriteAction
			if err := json.Unmarshal(message.Params, &action); err != nil {
				panic("invalid $pal/writeFile params: " + err.Error())
			}
			flush()
			segments = append(segments, segment{palWrite: &action})
			continue
		}
		if err := protocol.WriteMessage(&current.input, message); err != nil {
			panic("frame message: " + err.Error())
		}
	}
	flush()
	return segments
}

// applyPalWrite materializes a driver-side file write through PAL only
// (MkdirAll + WriteFile — no os.* escape hatch), sequenced between LSP
// batches. This is the approved mid-session on-disk write action (mirrors
// Java's Files.writeString mid-test, ProjectServiceTest.java:293-294).
func applyPalWrite(t *testing.T, platform pal.Platform, root string, action *palWriteAction) {
	t.Helper()
	if root == "" || action.Path == "" {
		t.Fatalf("palWrite needs a fixture root and path (path=%q root=%q)", action.Path, root)
	}
	abs := root + "/" + action.Path
	if err := platform.FS.MkdirAll(path.Dir(abs), 0o755); err != nil {
		t.Fatalf("palWrite MkdirAll %s: %v", path.Dir(abs), err)
	}
	if err := platform.FS.WriteFile(abs, []byte(action.Content)); err != nil {
		t.Fatalf("palWrite WriteFile %s: %v", abs, err)
	}
}

func fixtureMessages(rawMessages []json.RawMessage, source, root string) ([]protocol.Message, error) {
	replacement, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	messages := make([]protocol.Message, 0, len(rawMessages))
	for _, raw := range rawMessages {
		s := string(raw)
		s = strings.ReplaceAll(s, `"${source}"`, string(replacement))
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

// materializeSources writes a multi-file project tree onto an isolated temp
// root through PAL (MkdirAll + WriteFile — no os.* escape hatch) so build
// projects load via config_creator's ReadDir/Stat/ReadFile through palFS. The
// returned root path is substituted into fixture messages as ${root}.
func materializeSources(t *testing.T, platform pal.Platform, sources map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range sources {
		abs := root + "/" + rel
		if err := platform.FS.MkdirAll(path.Dir(abs), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", path.Dir(abs), err)
		}
		if err := platform.FS.WriteFile(abs, []byte(content)); err != nil {
			t.Fatalf("WriteFile %s: %v", abs, err)
		}
	}
	return root
}

func normalizeMessages(messages []json.RawMessage) []any {
	normalized := make([]any, len(messages))
	for index, message := range messages {
		_ = json.Unmarshal(message, &normalized[index])
	}
	return normalized
}

func formatMessages(messages []json.RawMessage) string {
	formatted, _ := json.MarshalIndent(messages, "", "  ")
	return string(formatted)
}
