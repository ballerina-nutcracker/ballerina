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
	"os"

	"github.com/ballerina-nutcracker/ballerina/ls/core/compile"
	"github.com/ballerina-nutcracker/ballerina/ls/core/event"
	"github.com/ballerina-nutcracker/ballerina/ls/core/observability"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/server"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"

	"github.com/spf13/cobra"
)

// stdioTransport adapts the process stdin/stdout pair into a single
// protocol.Transport (io.Reader + io.Writer) for the LSP server.
type stdioTransport struct{}

func (stdioTransport) Read(p []byte) (int, error)  { return os.Stdin.Read(p) }
func (stdioTransport) Write(p []byte) (int, error) { return os.Stdout.Write(p) }

var startLangServerCmd = &cobra.Command{
	Use:   "start-language-server",
	Short: "Start the Ballerina language server over stdio (LSP)",
	Long: `	Start the Ballerina language server, speaking the Language Server
	Protocol (LSP) over standard input/output. Editors and extensions
	launch this command and communicate via JSON-RPC framed with
	Content-Length headers.`,
	Args: cobra.NoArgs,
	RunE: startLangServer,
}

func startLangServer(cmd *cobra.Command, args []string) error {
	platform, cleanup := palnative.NewPlatform()
	defer cleanup()

	logger := observability.New(platform.IO, platform.OS)
	bus := event.New(event.WithLogf(func(msg string, args ...any) { logger.Warn(msg, args...) }))
	projects := workspace.New(platform, bus)
	compiler := compile.New(projects, bus)
	srv := server.New(stdioTransport{}, projects, compiler, bus, server.WithLogger(logger))

	// Bounded shutdown drain (branch 9 ordering): the workspace marks every
	// in-flight generation superseded; the compile engine then stops its
	// debounce timers, bounded-waits for in-flight compiles to finish (their
	// results gated out by the closed flag), and bus.Flush()es the CRITICAL
	// delivery channels; the bus closes last so that Flush drains before close.
	// Defers run LIFO, so execution order is projects → compiler → bus.
	defer bus.Close()
	defer compiler.Shutdown()
	defer projects.Shutdown(context.Background())
	return srv.Serve()
}
