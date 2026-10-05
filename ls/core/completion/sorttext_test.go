// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package completion

import "testing"

func TestRelevanceSortText(t *testing.T) {
	neutral := neutralRelevance
	if got, want := neutral, relevance(127); got != want {
		t.Errorf("neutralRelevance = %d, want %d", got, want)
	}
	if got, want := neutral.sortText(), "128"; got != want {
		t.Errorf("neutral sortText() = %q, want %q", got, want)
	}

	higher := neutral.add(1)
	if got, want := higher.sortText(), "127"; got != want {
		t.Errorf("higher sortText() = %q, want %q", got, want)
	}
	lower := neutral.add(-1)
	if got, want := lower.sortText(), "129"; got != want {
		t.Errorf("lower sortText() = %q, want %q", got, want)
	}
	if got, want := neutral.sortText(), "128"; got != want {
		t.Errorf("add mutated neutral: sortText() = %q, want %q", got, want)
	}

	if got, want := relevance(255).add(1).sortText(), "000"; got != want {
		t.Errorf("upper saturation sortText() = %q, want %q", got, want)
	}
	if got, want := relevance(0).add(-1).sortText(), "255"; got != want {
		t.Errorf("lower saturation sortText() = %q, want %q", got, want)
	}
	if got, want := neutral.sortText(), neutralRelevance.sortText(); got != want {
		t.Errorf("equal relevance sort texts differ: %q != %q", got, want)
	}
}
