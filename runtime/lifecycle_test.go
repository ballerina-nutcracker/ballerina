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

package runtime_test

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "ballerina-lang-go/lib/rt"
	"ballerina-lang-go/platform/pal"
	"ballerina-lang-go/projects"
	"ballerina-lang-go/runtime"
	"ballerina-lang-go/runtime/extern"
	"ballerina-lang-go/values"
)

// This defines tests that validate exit status and signal handling since we can't validate those
// via the corpus test harness

const lifecycleTestSource = `
import ballerina/io;

class ListenerOne {
    public function attach(service object {} svc, () attachPoint = ()) returns error? {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(service object {} svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
        io:println("start:one");
    }

    public function gracefulStop() returns error? {
        io:println("graceful:one");
    }

    public function immediateStop() returns error? {
        io:println("immediate:one");
    }
}

class ListenerTwo {
    public function attach(service object {} svc, () attachPoint = ()) returns error? {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(service object {} svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
        io:println("start:two");
    }

    public function gracefulStop() returns error? {
        io:println("graceful:two");
    }

    public function immediateStop() returns error? {
        io:println("immediate:two");
    }
}

listener ListenerOne l1 = new ();
listener ListenerTwo l2 = new ();

service on l1 {
}

service on l2 {
}
`

func TestLifecycleGracefulStopSignal(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt := newLifecycleTestRuntime(t, lifecycleTestSource, pal)

	rt.Listen()
	pal.Send(palSignalGracefulStop)
	code := readExitStatus(t, rt)

	if code != 130 {
		t.Fatalf("expected graceful stop exit code 130, got %d", code)
	}
	if got, want := pal.Stdout(), "start:one\nstart:two\ngraceful:one\ngraceful:two\n"; got != want {
		t.Fatalf("unexpected stdout: got %q, want %q", got, want)
	}
}

func TestLifecycleImmediateStopSignal(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt := newLifecycleTestRuntime(t, lifecycleTestSource, pal)

	rt.Listen()
	pal.Send(palSignalImmediateStop)
	code := readExitStatus(t, rt)

	if code != 131 {
		t.Fatalf("expected immediate stop exit code 131, got %d", code)
	}
	if got, want := pal.Stdout(), "start:one\nstart:two\nimmediate:one\nimmediate:two\n"; got != want {
		t.Fatalf("unexpected stdout: got %q, want %q", got, want)
	}
}

func TestLifecycleInitFailureStopsRuntime(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt, err := initLifecycleTestRuntime(t, lifecycleTestSource+`

public function main() returns error? {
    return error("init failed");
}
`, pal)
	if err == nil {
		t.Fatal("expected init error")
	}

	rt.Listen()
	code := readExitStatus(t, rt)

	if code != 1 {
		t.Fatalf("expected init failure exit code 1, got %d", code)
	}
	if got, want := pal.Stdout(), "graceful:one\ngraceful:two\n"; got != want {
		t.Fatalf("unexpected stdout: got %q, want %q", got, want)
	}
}

func TestLifecycleOnGracefulStopHandlers(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt := newLifecycleTestRuntime(t, `
import ballerina/lang.runtime;
`+lifecycleTestSource+`
public function stopOne() returns error? {
    io:println("handler:one");
}

public function stopTwo() returns error? {
    io:println("handler:two");
}

public function main() {
    runtime:onGracefulStop(stopOne);
    runtime:onGracefulStop(stopTwo);
}
`, pal)

	rt.Listen()
	pal.Send(palSignalGracefulStop)
	code := readExitStatus(t, rt)

	if code != 130 {
		t.Fatalf("expected graceful stop exit code 130, got %d", code)
	}
	if got, want := pal.Stdout(), "start:one\nstart:two\ngraceful:one\ngraceful:two\nhandler:two\nhandler:one\n"; got != want {
		t.Fatalf("unexpected stdout: got %q, want %q", got, want)
	}
}

func TestLifecycleOnGracefulStopWithoutListenersExitStatus(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt := newLifecycleTestRuntime(t, `
import ballerina/lang.runtime;

public function stopHandler() returns error? {
}

public function main() {
    runtime:onGracefulStop(stopHandler);
}
`, pal)

	rt.Listen()
	code := readExitStatus(t, rt)

	if code != 0 {
		t.Fatalf("expected successful exit code 0, got %d", code)
	}
}

func TestLifecycleOnGracefulStopWithoutListenersHandlerError(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt := newLifecycleTestRuntime(t, `
import ballerina/lang.runtime;

public function stopHandler() returns error? {
    return error("stop failed");
}

public function main() {
    runtime:onGracefulStop(stopHandler);
}
`, pal)

	rt.Listen()
	code := readExitStatus(t, rt)

	if code == 0 {
		t.Fatal("expected non-zero exit code when graceful stop handler fails")
	}
}

func TestLifecycleOnGracefulStopWithoutListenersAfterInitFailure(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt, err := initLifecycleTestRuntime(t, `
import ballerina/lang.runtime;

public function stopHandler() returns error? {
}

public function main() returns error? {
    runtime:onGracefulStop(stopHandler);
    return error("init failed");
}
`, pal)
	if err == nil {
		t.Fatal("expected init error")
	}

	rt.Listen()
	code := readExitStatus(t, rt)

	if code != 1 {
		t.Fatalf("expected init failure exit code 1, got %d", code)
	}
}

func TestLifecycleOnGracefulStopExternHandler(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt, err := initLifecycleTestRuntimeWithExterns(t, `
import ballerina/lang.runtime;
`+lifecycleTestSource+`
public function externalStop() returns error? = external;

public function main() {
    runtime:onGracefulStop(externalStop);
}
`, pal, []lifecycleExtern{{
		funcName: "externalStop",
		impl: func(_ *extern.Context, _ []values.BalValue) (values.BalValue, error) {
			_, _ = pal.stdout.Write([]byte("handler:extern\n"))
			return nil, nil
		},
	}})
	if err != nil {
		t.Fatal(err)
	}

	rt.Listen()
	pal.Send(palSignalGracefulStop)
	code := readExitStatus(t, rt)

	if code != 130 {
		t.Fatalf("expected graceful stop exit code 130, got %d", code)
	}
	if got, want := pal.Stdout(), "start:one\nstart:two\ngraceful:one\ngraceful:two\nhandler:extern\n"; got != want {
		t.Fatalf("unexpected stdout: got %q, want %q", got, want)
	}
}

func TestLifecycleOnGracefulStopAfterListenReportsCurrentState(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt := newLifecycleTestRuntime(t, `
import ballerina/lang.runtime;
`+lifecycleTestSource+`
public function stopHandler() returns error? {
}

public function registerAfterListen() {
    runtime:onGracefulStop(stopHandler);
}
`, pal)

	rt.Listen()
	fn, ok := runtime.LookupFunction(rt, "testorg", "lifecycletest", "registerAfterListen")
	if !ok {
		t.Fatal("failed to lookup registerAfterListen")
	}
	recovered := invokeAndRecover(rt, fn)
	if recovered == nil {
		t.Fatal("expected runtime:onGracefulStop after listen to fail")
	}
	message := fmt.Sprint(recovered)
	if !strings.Contains(message, "registering graceful stop listeners outside of module init not supported") {
		t.Fatalf("expected outside-init failure, got %q", message)
	}
}

func TestLifecycleOnGracefulStopAfterInitializationFailsFast(t *testing.T) {
	pal := newLifecycleTestPal(t)
	rt := newLifecycleTestRuntime(t, `
import ballerina/lang.runtime;

class RegisteringListener {
    public function attach(service object {} svc, () attachPoint = ()) returns error? {
        var _ = svc;
        var _ = attachPoint;
    }

    public function detach(service object {} svc) returns error? {
        var _ = svc;
    }

    public function 'start() returns error? {
        runtime:onGracefulStop(stopHandler);
    }

    public function gracefulStop() returns error? {
    }

    public function immediateStop() returns error? {
    }
}

public function stopHandler() returns error? {
}

listener RegisteringListener l = new ();

service on l {
}
`, pal)

	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		rt.Listen()
	}()

	select {
	case recovered := <-done:
		if recovered == nil {
			t.Fatal("expected runtime:onGracefulStop outside initialization to fail fast")
		}
		message := fmt.Sprint(recovered)
		if !strings.Contains(message, "can't register graceful stop listeners during state transitions") {
			t.Fatalf("expected lifecycle-busy failure, got %q", message)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runtime:onGracefulStop outside initialization to fail fast")
	}
}

func invokeAndRecover(rt *runtime.Runtime, fn any) (recovered any) {
	defer func() {
		recovered = recover()
	}()
	_, err := runtime.InvokeFunction(rt, fn, nil)
	return err
}

func newLifecycleTestRuntime(t *testing.T, source string, platform *lifecycleTestPal) *runtime.Runtime {
	t.Helper()

	rt, err := initLifecycleTestRuntime(t, source, platform)
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func initLifecycleTestRuntime(t *testing.T, source string, platform *lifecycleTestPal) (*runtime.Runtime, error) {
	t.Helper()
	return initLifecycleTestRuntimeWithExterns(t, source, platform, nil)
}

type lifecycleExtern struct {
	funcName string
	impl     extern.NativeFunc
}

func initLifecycleTestRuntimeWithExterns(t *testing.T, source string, platform *lifecycleTestPal, externs []lifecycleExtern) (*runtime.Runtime, error) {
	t.Helper()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Ballerina.toml"), `[package]
org = "testorg"
name = "lifecycletest"
version = "0.1.0"
`)
	writeFile(t, filepath.Join(dir, "main.bal"), source)

	ballerinaEnvFs, err := ballerinaEnvFS()
	if err != nil {
		t.Fatal(err)
	}
	result, err := projects.Load(os.DirFS(dir), ".", projects.ProjectLoadConfig{BallerinaEnvFs: ballerinaEnvFs})
	if err != nil {
		t.Fatal(err)
	}
	compilation := result.Project().CurrentPackage().Compilation()
	if result.Diagnostics().HasErrors() || compilation.DiagnosticResult().HasErrors() {
		t.Fatalf("lifecycle test project has diagnostics: load=%v compile=%v", result.Diagnostics().Errors(), compilation.DiagnosticResult().Errors())
	}
	pkgs := projects.NewBallerinaBackend(compilation).BIRPackages()
	if len(pkgs) == 0 {
		t.Fatal("compilation succeeded but produced no BIR packages")
	}

	rt := runtime.NewRuntime(platform.Platform(), result.Project().Environment().TypeEnv())
	for _, e := range externs {
		runtime.RegisterExternFunction(rt, "testorg", "lifecycletest", e.funcName, e.impl)
	}
	for _, pkg := range pkgs {
		if err := rt.Init(*pkg); err != nil {
			return rt, err
		}
	}
	return rt, nil
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func ballerinaEnvFS() (fs.FS, error) {
	if v := os.Getenv(projects.BallerinaEnvVar); v != "" {
		return os.DirFS(v), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return os.DirFS(filepath.Join(home, projects.UserHomeDirName)), nil
}

func readExitStatus(t *testing.T, rt *runtime.Runtime) uint8 {
	t.Helper()
	select {
	case code := <-rt.ExitStatus:
		return code
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for runtime exit status")
	}
	return 0
}

type lifecycleTestPal struct {
	stdout  bytes.Buffer
	stderr  bytes.Buffer
	signals chan pal.Signal
}

const (
	palSignalGracefulStop  = pal.GracefulStop
	palSignalImmediateStop = pal.ImmediateStop
)

func newLifecycleTestPal(t *testing.T) *lifecycleTestPal {
	t.Helper()
	p := &lifecycleTestPal{signals: make(chan pal.Signal, 4)}
	t.Cleanup(func() { close(p.signals) })
	return p
}

func (p *lifecycleTestPal) Platform() pal.Platform {
	return pal.Platform{
		IO: pal.IO{
			Stdout: p.stdout.Write,
			Stderr: p.stderr.Write,
		},
		FS: pal.FS{
			ReadFile: func(path string) ([]byte, error) {
				return nil, &fs.PathError{Op: "open", Path: path, Err: fs.ErrNotExist}
			},
			OpenReadable: func(path string) (io.ReadCloser, error) {
				return os.Open(path)
			},
			OpenWritable: func(path string, appendMode bool) (io.WriteCloser, error) {
				flag := os.O_CREATE | os.O_WRONLY
				if appendMode {
					flag |= os.O_APPEND
				} else {
					flag |= os.O_TRUNC
				}
				return os.OpenFile(path, flag, 0o644)
			},
		},
		HTTP: pal.HTTP{
			NewClient: func(_ pal.ClientConfig) pal.HTTPClient { return nil },
		},
		Signals: pal.SignalSource{Signals: p.signals},
	}
}

func (p *lifecycleTestPal) Send(signal pal.Signal) {
	p.signals <- signal
}

func (p *lifecycleTestPal) Stdout() string {
	return p.stdout.String()
}

func (p *lifecycleTestPal) String() string {
	return fmt.Sprintf("stdout=%q stderr=%q", p.stdout.String(), p.stderr.String())
}
