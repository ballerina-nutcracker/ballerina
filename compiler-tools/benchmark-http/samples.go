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
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
)

type sampleMetadata struct {
	Type    string         `json:"type"`
	Version int            `json:"version"`
	BaseSHA string         `json:"base_sha"`
	HeadSHA string         `json:"head_sha"`
	Suite   string         `json:"suite"`
	Phase   string         `json:"phase"`
	Cases   []string       `json:"cases"`
	Counts  map[string]int `json:"counts"`
	Unit    string         `json:"unit"`
}

type sampleEvent struct {
	Type     string             `json:"type"`
	CaseID   string             `json:"case_id"`
	Pair     int                `json:"pair"`
	Role     string             `json:"role"`
	Position int                `json:"position"`
	Value    float64            `json:"value"`
	Unit     string             `json:"unit"`
	Metrics  map[string]float64 `json:"metrics,omitempty"`
}

type sampleWriter struct {
	file    *os.File
	encoder *json.Encoder
}

func resolveSampleSHA(worktree string) (string, error) {
	out, err := exec.Command("git", "-C", worktree, "rev-parse", "--verify", "HEAD^{commit}").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func newSampleWriter(path string, metadata sampleMetadata) (*sampleWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	w := &sampleWriter{file: f, encoder: json.NewEncoder(f)}
	if err := w.encoder.Encode(metadata); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return nil, err
	}
	return w, nil
}

func (w *sampleWriter) write(event sampleEvent) error {
	if event.Value <= 0 || math.IsNaN(event.Value) || math.IsInf(event.Value, 0) {
		return fmt.Errorf("invalid primary sample %v", event.Value)
	}
	event.Type = "sample"
	if err := w.encoder.Encode(event); err != nil {
		return err
	}
	return w.file.Sync()
}

func (w *sampleWriter) completeCase(id string) error {
	if err := w.encoder.Encode(map[string]string{"type": "case_complete", "case_id": id}); err != nil {
		return err
	}
	return w.file.Sync()
}

func (w *sampleWriter) close(complete bool) error {
	if complete {
		if err := w.encoder.Encode(map[string]string{"type": "complete"}); err != nil {
			_ = w.file.Close()
			return err
		}
	}
	return w.file.Close()
}
