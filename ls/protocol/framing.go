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

package protocol

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Transport carries framed JSON-RPC messages in both directions.
type Transport interface {
	io.Reader
	io.Writer
}

// ReadMessage reads one Content-Length-framed JSON-RPC message.
func ReadMessage(reader *bufio.Reader) (Message, error) {
	payload, err := ReadFrame(reader)
	if err != nil {
		return Message{}, err
	}
	var message Message
	if err := json.Unmarshal(payload, &message); err != nil {
		return Message{}, err
	}
	return message, nil
}

// ReadFrame reads a complete Content-Length-framed body without decoding JSON.
func ReadFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := -1
	started := false
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && (started || len(line) > 0) {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		started = true
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			continue
		}
		contentLength, err = strconv.Atoi(strings.TrimSpace(value))
		if err != nil || contentLength < 0 {
			return nil, fmt.Errorf("invalid Content-Length header")
		}
	}
	if contentLength < 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return payload, nil
}

// WriteMessage writes message as Content-Length-framed JSON.
// Callers must serialize writes to the same stream.
func WriteMessage(writer io.Writer, message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return err
	}
	_, err = writer.Write(payload)
	return err
}
