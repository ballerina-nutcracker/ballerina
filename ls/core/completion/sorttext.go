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

import "fmt"

type relevance uint8

const maxRelevance = ^uint8(0)
const neutralRelevance relevance = relevance(maxRelevance / 2)

func (r relevance) add(delta int8) relevance {
	score := int16(r) + int16(delta)
	if score < 0 {
		return 0
	}
	if score > int16(maxRelevance) {
		return relevance(maxRelevance)
	}
	return relevance(score)
}

func (r relevance) sortText() string {
	return fmt.Sprintf("%03d", maxRelevance-uint8(r))
}
