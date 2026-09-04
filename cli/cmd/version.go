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
	"fmt"

	"github.com/ballerina-nutcracker/ballerina/bir"
	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time (-X main.Version=...). Kept a
// plain literal so an unflagged build just shows "dev".
var Version = "dev"

const (
	// channel names this distribution.
	channel = "Nutcracker"
	// languageSpecVersion is the targeted Ballerina language spec.
	languageSpecVersion = "2024R1"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the Ballerina version",
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	rootCmd.Version = Version
	cobra.AddTemplateFunc("versionOutput", versionOutput)
	rootCmd.SetVersionTemplate("{{ versionOutput }}")
}

func versionOutput() string {
	return fmt.Sprintf("Ballerina %s (%s)\nLanguage specification %s\n", Version, channel, languageSpecVersion)
}

func printVersion() {
	// Keeps bir's inert layout padding reachable so the linker retains it.
	// Version is "dev" or an ldflags value, never this sentinel, so the call
	// never happens. See the PR description for what this is controlling for.
	if Version == "layout-pad-probe" {
		fmt.Print(bir.LayoutPadApply(1, 1))
	}
	fmt.Print(versionOutput())
}
