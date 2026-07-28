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
	"strconv"
	"strings"
)

// wrkMetrics holds the numbers extracted from one `wrk --latency` run.
// Latency values are normalised to milliseconds.
type wrkMetrics struct {
	throughput float64 // requests/sec
	avgMs      float64
	stdevMs    float64
	maxMs      float64
	p99Ms      float64
	errPct     float64
}

// parseWrk extracts throughput, latency stats and an error rate from the
// stdout of `wrk --latency`. It mirrors performance/scripts/parse_wrk.sh.
func parseWrk(out string) (wrkMetrics, error) {
	var m wrkMetrics
	var totalReq, sockErr, httpErr float64
	haveThroughput := false

	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		switch {
		case f[0] == "Latency" && len(f) >= 4 && isDurationToken(f[1]):
			// Thread Stats latency row: "Latency  <avg> <stdev> <max> <+/-stdev>"
			m.avgMs, _ = toMs(f[1])
			m.stdevMs, _ = toMs(f[2])
			m.maxMs, _ = toMs(f[3])
		case f[0] == "99%" && len(f) >= 2:
			m.p99Ms, _ = toMs(f[1])
		case f[0] == "Requests/sec:" && len(f) >= 2:
			if v, err := strconv.ParseFloat(f[1], 64); err == nil {
				m.throughput = v
				haveThroughput = true
			}
		case len(f) >= 3 && f[1] == "requests" && f[2] == "in":
			// "<n> requests in <dur>, <bytes> read"
			totalReq, _ = strconv.ParseFloat(f[0], 64)
		case f[0] == "Socket" && len(f) >= 2 && f[1] == "errors:":
			sockErr = sumTrailingInts(f)
		case f[0] == "Non-2xx" && len(f) >= 2:
			// "Non-2xx or 3xx responses: <n>"
			httpErr, _ = strconv.ParseFloat(f[len(f)-1], 64)
		}
	}

	if !haveThroughput {
		return m, fmt.Errorf("no Requests/sec line found in wrk output")
	}
	if totalReq > 0 {
		m.errPct = (sockErr + httpErr) / totalReq * 100
	}
	return m, nil
}

// sumTrailingInts adds every integer token on a line (used for the
// "Socket errors: connect X, read Y, write Z, timeout W" line).
func sumTrailingInts(fields []string) float64 {
	var sum float64
	for _, tok := range fields {
		tok = strings.TrimRight(tok, ",")
		if v, err := strconv.ParseFloat(tok, 64); err == nil {
			sum += v
		}
	}
	return sum
}

// isDurationToken reports whether tok looks like a wrk latency value such as
// "1.76ms" (a leading digit and a trailing unit), so the "Latency Distribution"
// header line is not mistaken for the Thread Stats latency row.
func isDurationToken(tok string) bool {
	if tok == "" || (tok[0] < '0' || tok[0] > '9') {
		return false
	}
	_, err := toMs(tok)
	return err == nil
}

// toMs converts a wrk duration token (e.g. "850.00us", "1.76ms", "1.20s",
// "2.00m") to milliseconds.
func toMs(tok string) (float64, error) {
	tok = strings.TrimSpace(tok)
	i := len(tok)
	for i > 0 {
		c := tok[i-1]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			i--
		} else {
			break
		}
	}
	num := tok[:i]
	unit := tok[i:]
	v, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse duration %q: %w", tok, err)
	}
	switch unit {
	case "us":
		return v / 1000, nil
	case "ms", "":
		return v, nil
	case "s":
		return v * 1000, nil
	case "m":
		return v * 60000, nil
	case "h":
		return v * 3600000, nil
	default:
		return 0, fmt.Errorf("unknown duration unit %q in %q", unit, tok)
	}
}
