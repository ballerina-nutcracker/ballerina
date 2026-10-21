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

// Port of jBallerina's FailSafeUtils: errors that failSafe tolerates are logged to the console and to a
// file instead of failing the call.

package native

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ballerina-nutcracker/ballerina/runtime"
	"github.com/ballerina-nutcracker/ballerina/values"
)

type failSafeLogger struct {
	rt            *runtime.Runtime
	console       bool
	includeSource bool
	file          *fileOutput
}

type fileOutput struct {
	path        string
	contentType string
	overwrite   bool
	overwritten bool
}

// newFailSafeLogger returns nil when failSafe is not configured. Console logs are printed only when
// withConsole is set: jBallerina never prints them for transform, parseList and union element types.
func newFailSafeLogger(rt *runtime.Runtime, options *values.Map, withConsole bool) *failSafeLogger {
	if options == nil {
		return nil
	}
	logger := &failSafeLogger{
		rt:            rt,
		console:       withConsole && get(options, "enableConsoleLogs") == true,
		includeSource: get(options, "includeSourceDataInConsole") == true,
	}
	if mode, ok := get(options, "fileOutputMode").(*values.Map); ok {
		path, _ := get(mode, "filePath").(string)
		contentType, _ := get(mode, "contentType").(string)
		writeOption, _ := get(mode, "fileWriteOption").(string)
		logger.file = &fileOutput{path: path, contentType: contentType, overwrite: writeOption == "OVERWRITE"}
	}
	return logger
}

// log records a tolerated error at a 1-based row and column. Only a failure to write the log file is
// returned.
func (l *failSafeLogger) log(err error, row, column int, offendingRow string) error {
	if l.console {
		l.printToConsole(err, row, column, offendingRow)
	}
	if l.file != nil {
		return l.writeToFile(err, row, column, offendingRow)
	}
	return nil
}

func (l *failSafeLogger) printToConsole(err error, row, column int, offendingRow string) {
	message := fmt.Sprintf("CSV parse error at row %d, column %d: %s", row, column, err.Error())
	var source values.BalValue
	if l.includeSource {
		source = javaTrim(offendingRow)
	}
	ctx := l.rt.AcquirePooledContext()
	defer l.rt.ReleasePooledContext(ctx)
	if handle, ok := ctx.LookupFunction(orgName, moduleName, "printError"); ok {
		_, _ = ctx.InvokeFunction(handle, []values.BalValue{message, source})
	}
}

func (l *failSafeLogger) writeToFile(err error, row, column int, offendingRow string) error {
	fsys := l.rt.Platform().FS
	path := l.file.path
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if mkErr := fsys.MkdirAll(dir); mkErr != nil {
			return fileIOError(fmt.Sprintf("Failed to create log file at: %s. Caused by: %s\n", path, mkErr))
		}
	}
	if _, statErr := fsys.Stat(path); errors.Is(statErr, fs.ErrNotExist) {
		if createErr := fsys.CreateFile(path); createErr != nil {
			return fileIOError(fmt.Sprintf("Failed to create log file at: %s. Caused by: %s\n", path, createErr))
		}
	}
	if l.file.overwrite && !l.file.overwritten {
		if writeErr := fsys.WriteFile(path, nil); writeErr != nil {
			return fileIOError(fmt.Sprintf("Failed to overwrite log file at: %s. Caused by: %s", path, writeErr))
		}
		l.file.overwritten = true
	}
	content := offendingRow
	if l.file.contentType != "RAW" {
		content = l.jsonLog(err, row, column, offendingRow)
	}
	if appendErr := fsys.AppendFile(path, []byte(javaTrim(content)+"\n")); appendErr != nil {
		return fileIOError(fmt.Sprintf("Failed to write log file at: %s. Caused by: %s", path, appendErr))
	}
	return nil
}

// jsonLog builds one JSON log line. jBallerina concatenates the values without escaping them; they are
// escaped here so that every line is valid JSON.
func (l *failSafeLogger) jsonLog(err error, row, column int, offendingRow string) string {
	var sb strings.Builder
	sb.WriteString(`{"time":`)
	sb.WriteString(jsonString(javaInstantString(l.rt.Platform().Time.Now())))
	sb.WriteString(`,"location":{"row":`)
	sb.WriteString(strconv.Itoa(row))
	sb.WriteString(`,"column":`)
	sb.WriteString(strconv.Itoa(column))
	sb.WriteString(`},`)
	if l.file.contentType == "RAW_AND_METADATA" {
		sb.WriteString(`"offendingRow":`)
		sb.WriteString(jsonString(javaTrim(offendingRow)))
		sb.WriteString(`,`)
	}
	sb.WriteString(`"message":`)
	sb.WriteString(jsonString(err.Error()))
	sb.WriteString(`}`)
	return sb.String()
}

func jsonString(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&sb, `\u%04x`, r)
				continue
			}
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

// javaInstantString renders t as java.time.Instant.toString does: UTC, with the fraction of a second in
// groups of three digits and omitted when zero.
func javaInstantString(t time.Time) string {
	t = t.UTC()
	base := t.Format("2006-01-02T15:04:05")
	nanos := t.Nanosecond()
	switch {
	case nanos == 0:
		return base + "Z"
	case nanos%1_000_000 == 0:
		return fmt.Sprintf("%s.%03dZ", base, nanos/1_000_000)
	case nanos%1_000 == 0:
		return fmt.Sprintf("%s.%06dZ", base, nanos/1_000)
	}
	return fmt.Sprintf("%s.%09dZ", base, nanos)
}
