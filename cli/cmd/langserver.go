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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var startLangServerCmd = &cobra.Command{
	Use:   "start-language-server",
	Short: "Start the Ballerina language server over stdio (LSP)",
	Long: `	Start the Ballerina language server, speaking the Language Server
	Protocol (LSP) over standard input/output. Editors and extensions
	launch this command and communicate via JSON-RPC framed with
	Content-Length headers.`,
	Args:   cobra.NoArgs,
	Hidden: true,
	RunE:   startLangServer,
}

func startLangServer(cmd *cobra.Command, args []string) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating bal for language server: %w", err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("resolving bal for language server: %w", err)
	}
	name := "bal-language-server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	child := exec.Command(filepath.Join(filepath.Dir(executable), name))
	child.Stdin, child.Stdout, child.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := child.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", name, err)
	}
	if err := child.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code := exitErr.ExitCode()
			if code < 0 {
				code = 1
			}
			os.Exit(code)
		}
		return fmt.Errorf("waiting for %s: %w", name, err)
	}
	return nil
}
