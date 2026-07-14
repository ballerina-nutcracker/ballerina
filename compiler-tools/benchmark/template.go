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

const htmlTemplate = `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>{{ .Title }}</title>
	<link rel="preconnect" href="https://fonts.googleapis.com">
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
	<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:ital,wght@0,100..800;1,100..800&display=swap" rel="stylesheet">
	<style>
	*, *::before, *::after {
		box-sizing: border-box;
		margin: 0;
		padding: 0;
	}
	:root {
		--black: #000000;
		--white: #ffffff;
		--border: #e7e3e4;
		--dim: #999999;
	}
	html, body {
		background: var(--white);
		color: var(--black);
		font-family: 'JetBrains Mono', monospace;
		font-size: 14px;
		line-height: 1.6;
		-webkit-font-smoothing: antialiased;
	}
	body {
		padding: 48px 40px;
		max-width: 1080px;
		margin: 0 auto;
	}
	header {
		padding-bottom: 24px;
		margin-bottom: 40px;
	}
	.header-eyebrow {
		font-size: 10px;
		font-weight: 500;
		color: var(--dim);
		letter-spacing: 0.1em;
		text-transform: uppercase;
		margin-bottom: 8px;
	}
	.header-refs {
		font-size: 20px;
		font-weight: 700;
		letter-spacing: -0.02em;
	}
	.header-refs .sep {
		font-weight: 300;
		color: var(--dim);
		margin: 0 10px;
	}
	.header-meta {
		margin-top: 8px;
		font-size: 11px;
		color: var(--dim);
		display: flex;
		gap: 20px;
		flex-wrap: wrap;
	}
	.section-label {
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.12em;
		text-transform: uppercase;
		color: var(--dim);
		margin-bottom: 16px;
	}
	.table-wrap {
		border: 1px solid var(--border);
		overflow-x: auto;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		min-width: 860px;
	}
	.col-sep {
		border-left: 1px solid var(--border);
	}
	thead tr.row-groups th,
	thead tr.row-subs th {
		padding: 8px 16px;
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.1em;
		text-transform: uppercase;
		color: var(--dim);
		border-bottom: 1px solid var(--border);
		white-space: nowrap;
	}
	thead tr.row-subs th {
		font-weight: 500;
		letter-spacing: 0.08em;
		text-align: right;
	}
	thead th.align-left  { text-align: left; }
	thead th.align-right { text-align: right; }
	tbody tr {
		border-bottom: 1px solid var(--border);
	}
	tbody tr:last-child {
		border-bottom: none;
	}
	tbody td {
		padding: 13px 16px;
		font-size: 12px;
		text-align: right;
		font-variant-numeric: tabular-nums;
		vertical-align: middle;
	}
	tbody td.align-left { text-align: left; }
	.metric {
		font-size: 16px;
		font-weight: 700;
		color: var(--black);
	}
	.delta-ratio {
		font-size: 16px;
		font-weight: 700;
		color: var(--black);
	}
	.delta-margin {
		font-size: 14px;
		font-weight: 400;
		color: var(--black);
	}
	.delta-winner {
		display: block;
		font-size: 12px;
		color: var(--black);
		letter-spacing: 0.04em;
		margin-top: 4px;
	}
	.not-available {
		color: var(--dim);
		font-style: italic;
		font-size: 11px;
	}
	</style>
</head>
<body>
	<header>
		<div class="header-eyebrow">{{ .Title }}</div>
		<div class="header-refs">
			<span>{{ .Report.BaseRef }}</span>
			<span class="sep">vs</span>
			<span>{{ .Report.HeadRef }}</span>
		</div>
		<div class="header-meta">
			<span>generated {{ .Report.Generated.Format "2006-01-02 15:04:05" }}</span>
		</div>
	</header>
	<div class="section-label">Results</div>
	<div class="table-wrap">
		<table>
			<thead>
				<tr class="row-groups">
					<th class="align-left" rowspan="2">CASE</th>
					<th class="col-sep" colspan="2">{{ .Report.BaseRef }}</th>
					<th class="col-sep" colspan="2">{{ .Report.HeadRef }}</th>
					<th class="col-sep align-right" rowspan="2">DELTA</th>
				</tr>
				<tr class="row-subs">
					<th class="col-sep">{{ .MeanLabel }}</th>
					<th class="col-sep">{{ .StddevLabel }}</th>
					<th class="col-sep">{{ .MeanLabel }}</th>
					<th class="col-sep">{{ .StddevLabel }}</th>
				</tr>
			</thead>
			<tbody>
				{{ range .Rows }}
				<tr>
					<td class="align-left">{{ .Label }}</td>
					<td class="col-sep">{{ if .Base }}<span class="metric">{{ .BaseMean }}</span>{{ else }}<span class="not-available">n/a</span>{{ end }}</td>
					<td class="col-sep">{{ if .Base }}<span class="metric">{{ .BaseStddev }}</span>{{ else }}<span class="not-available">n/a</span>{{ end }}</td>
					<td class="col-sep">{{ if .Head }}<span class="metric">{{ .HeadMean }}</span>{{ else }}<span class="not-available">n/a</span>{{ end }}</td>
					<td class="col-sep">{{ if .Head }}<span class="metric">{{ .HeadStddev }}</span>{{ else }}<span class="not-available">n/a</span>{{ end }}</td>
					<td class="col-sep">
						{{ if .DeltaAvailable }}
						<span class="delta-ratio">{{ .DeltaRatio }}×</span>
						<span class="delta-margin"> ± {{ .DeltaStddev }}</span>
						{{ if eq .DeltaWinnerRef "tie" }}
						<span class="delta-winner">tie</span>
						{{ else }}
						<span class="delta-winner">{{ .DeltaWinnerRef }} {{ $.WinnerVerb }}</span>
						{{ end }}
						{{ else }}
						<span class="not-available">n/a</span>
						{{ end }}
					</td>
				</tr>
				{{ end }}
			</tbody>
		</table>
	</div>
</body>
</html>`
