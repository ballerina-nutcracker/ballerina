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

package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
	"github.com/ballerina-nutcracker/ballerina/ls/server"
)

type stdioTransport struct{}

func (stdioTransport) Read(p []byte) (int, error)  { return os.Stdin.Read(p) }
func (stdioTransport) Write(p []byte) (int, error) { return os.Stdout.Write(p) }

func main() {
	if err := serve(); err != nil {
		fmt.Fprintln(os.Stderr, "ballerina language server:", err)
		os.Exit(1)
	}
}

func serve() error {
	return serveHost(stdioTransport{}, os.DirFS, os.Stderr)
}

// serveHost composes the server using host-owned transport, filesystem and log
// output. The caller retains ownership of the streams, including log output.
func serveHost(transport protocol.Transport, openDir func(string) fs.FS, logOutput io.Writer, opts ...workspace.Option) error {
	logger := observability.New(logOutput)
	bus := event.New(event.WithLogf(func(msg string, args ...any) { logger.Warn(msg, args...) }))
	projects := workspace.New(openDir, bus, opts...)
	compiler := compile.New(projects, bus, compile.WithLogger(logger))
	srv := server.New(transport, projects, compiler, bus, server.WithLogger(logger))

	// Supersede workspace generations before draining compiles; keep the bus
	// open until the compiler finishes flushing events. Defers run in reverse order.
	defer bus.Close()
	defer compiler.Shutdown()
	defer func() { _ = projects.Shutdown(context.Background()) }()
	return srv.Serve()
}
