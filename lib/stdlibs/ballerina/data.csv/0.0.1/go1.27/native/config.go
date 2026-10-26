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

package native

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ballerina-nutcracker/ballerina/values"
)

// config is jBallerina's CsvConfig: the options of one call. Unlike jBallerina, reading the options never
// writes back into the caller's options value, so reusing an options value gives the same result.
type config struct {
	allowDataProjection bool
	nilAsOptionalField  bool
	absentAsNilableType bool
	skipLines           values.BalValue
	outputWithHeaders   bool
	failSafe            *values.Map
	// enableConstraintValidation is read but constraints are not validated; see the package README.
	enableConstraintValidation bool

	delimiter     rune
	textEnclosure rune
	escapeChar    rune
	comment       rune
	locale        string
	encoding      string
	nilValue      *string
	// header is the 0-based line of the header row, or nil when there is no header row.
	header                       *int64
	customHeadersIfHeadersAbsent []string

	headerRows    int64
	customHeaders []string
	headerOrder   []string

	// stringConversion makes the traversal convert string values with FromString, as parseList does.
	stringConversion bool
}

func readOptions(opts *values.Map) *config {
	cfg := &config{
		delimiter:     ',',
		textEnclosure: '"',
		escapeChar:    '\\',
		comment:       '#',
		locale:        "en_US",
		encoding:      "UTF-8",
		header:        new(int64),
	}
	if projection, ok := get(opts, "allowDataProjection").(*values.Map); ok {
		cfg.allowDataProjection = true
		cfg.nilAsOptionalField = get(projection, "nilAsOptionalField") == true
		cfg.absentAsNilableType = get(projection, "absentAsNilableType") == true
	}
	cfg.skipLines = get(opts, "skipLines")
	cfg.outputWithHeaders = get(opts, "outputWithHeaders") == true
	cfg.enableConstraintValidation = get(opts, "enableConstraintValidation") != false
	cfg.failSafe, _ = get(opts, "failSafe").(*values.Map)
	if v, ok := get(opts, "delimiter").(string); ok {
		cfg.delimiter = firstRune(v)
	}
	if v, ok := get(opts, "textEnclosure").(string); ok {
		cfg.textEnclosure = firstRune(v)
	}
	if v, ok := get(opts, "escapeChar").(string); ok {
		cfg.escapeChar = firstRune(v)
	}
	if v, ok := get(opts, "comment").(string); ok {
		cfg.comment = firstRune(v)
	}
	if v, ok := get(opts, "locale").(string); ok {
		cfg.locale = v
	}
	if v, ok := get(opts, "encoding").(string); ok {
		cfg.encoding = v
	}
	if v, ok := get(opts, "nilValue").(string); ok {
		cfg.nilValue = &v
	}
	if opts != nil {
		if _, present := opts.Get("header"); present {
			cfg.header = nil
			if v, ok := get(opts, "header").(int64); ok {
				cfg.header = &v
			}
		}
	}
	cfg.customHeadersIfHeadersAbsent = stringList(get(opts, "customHeadersIfHeadersAbsent"))
	if v, ok := get(opts, "headerRows").(int64); ok {
		cfg.headerRows = v
	}
	cfg.customHeaders = stringList(get(opts, "customHeaders"))
	cfg.headerOrder = stringList(get(opts, "headerOrder"))
	return cfg
}

// unionMemberConfig is CsvConfig.createConfigOptionsForUnion: the parser options for reading the rows
// of a union element type as map<string> values before they are converted.
func (c *config) unionMemberConfig() *config {
	return &config{
		delimiter:     c.delimiter,
		textEnclosure: c.textEnclosure,
		escapeChar:    c.escapeChar,
		comment:       c.comment,
		locale:        c.locale,
		encoding:      c.encoding,
		nilValue:      c.nilValue,
		header:        c.header,
		failSafe:      c.failSafe,
	}
}

func (c *config) converter(t *typer) *converter {
	return &converter{t: t, nilValue: c.nilValue, locale: localeFor(c.locale)}
}

// skipSet is the 1-based data rows to skip, either listed or as an inclusive range; a range is not
// expanded so a huge range costs nothing.
type skipSet struct {
	rows       []int64
	start, end int64
	isRange    bool
}

func (s skipSet) contains(row int64) bool {
	if s.isRange {
		return row >= s.start && row <= s.end
	}
	return slices.Contains(s.rows, row)
}

// skipDataRows is CsvUtils.getSkipDataRows: the data rows to skip, from either a list of row numbers or a
// "start-end" range.
func (c *config) skipDataRows() (skipSet, error) {
	switch v := c.skipLines.(type) {
	case *values.List:
		rows := make([]int64, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			n, _ := v.Get(i).(int64)
			rows = append(rows, n)
		}
		return skipSet{rows: rows}, nil
	case string:
		return skipRange(v)
	}
	return skipSet{}, nil
}

func skipRange(spec string) (skipSet, error) {
	parts := strings.Split(spec, "-")
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) != 2 {
		return skipSet{}, skipLinesRangeError()
	}
	start, startErr := strconv.ParseInt(parts[0], 10, 32)
	end, endErr := strconv.ParseInt(parts[1], 10, 32)
	if startErr != nil || endErr != nil {
		return skipSet{}, skipLinesValueError()
	}
	if end < start {
		return skipSet{}, skipLinesRangeError()
	}
	return skipSet{start: start, end: end, isRange: true}, nil
}

func get(m *values.Map, key string) values.BalValue {
	if m == nil {
		return nil
	}
	v, _ := m.Get(key)
	return v
}

// stringList reads an optional string[] option; nil means the option is absent or `()`.
func stringList(v values.BalValue) []string {
	list, ok := v.(*values.List)
	if !ok {
		return nil
	}
	result := make([]string, 0, list.Len())
	for i := 0; i < list.Len(); i++ {
		s, _ := list.Get(i).(string)
		result = append(result, s)
	}
	return result
}

func firstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}
