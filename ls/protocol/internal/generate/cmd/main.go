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

// Command generate reads an LSP metamodel JSON file and its JSON schema and
// writes the generated Go protocol types into the directory specified by -out.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ballerina-nutcracker/ballerina/ls/protocol/internal/generate"
	"github.com/ballerina-nutcracker/ballerina/platform/palnative"
)

func main() {
	var (
		modelPath  string
		schemaPath string
		outDir     string
	)
	flag.StringVar(&modelPath, "model", "", "path to metaModel.json")
	flag.StringVar(&schemaPath, "schema", "", "path to metaModel.schema.json")
	flag.StringVar(&outDir, "out", "ls/protocol", "directory to write generated files")
	flag.Parse()

	if modelPath == "" || schemaPath == "" {
		log.Fatalf("-model and -schema are required")
	}

	modelJSON, err := os.ReadFile(modelPath)
	if err != nil {
		log.Fatalf("read metamodel: %v", err)
	}
	schemaJSON, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("read schema: %v", err)
	}

	pm, err := generate.Parse(modelJSON, schemaJSON)
	if err != nil {
		log.Fatalf("parse metamodel: %v", err)
	}
	if pm.Model.Version.Version != "3.18.0" {
		log.Fatalf("unexpected metamodel version: %s", pm.Model.Version.Version)
	}

	typesSrc, jsonSrc, err := generate.Generate(pm, modelJSON)
	if err != nil {
		log.Fatalf("generate: %v", err)
	}

	plat, _ := palnative.NewPlatform()
	typesPath := filepath.Join(outDir, generate.TypesFileName)
	jsonPath := filepath.Join(outDir, generate.JSONFileName)
	if err := plat.FS.WriteFile(typesPath, typesSrc); err != nil {
		log.Fatalf("write %s: %v", typesPath, err)
	}
	if err := plat.FS.WriteFile(jsonPath, jsonSrc); err != nil {
		log.Fatalf("write %s: %v", jsonPath, err)
	}
	fmt.Printf("generated %s and %s\n", typesPath, jsonPath)
}

func sha256sum(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
