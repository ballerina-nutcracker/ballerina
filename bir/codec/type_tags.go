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

package codec

// typeTag is the constant-value type discriminator persisted on the wire as int8.
// Values must match ast.TypeTags on the producing side; kept private here to
// avoid depending on the ast package from the codec.
type typeTag int8

const (
	typeTagInt        typeTag = 1
	typeTagByte       typeTag = 2
	typeTagFloat      typeTag = 3
	typeTagDecimal    typeTag = 4
	typeTagString     typeTag = 5
	typeTagBoolean    typeTag = 6
	typeTagNil        typeTag = 10
	typeTagSigned32   typeTag = 39
	typeTagSigned16   typeTag = 40
	typeTagSigned8    typeTag = 41
	typeTagUnsigned32 typeTag = 42
	typeTagUnsigned16 typeTag = 43
	typeTagUnsigned8  typeTag = 44
	typeTagCharString typeTag = 45
	typeTagMap        typeTag = 46
	typeTagTypedesc   typeTag = 47
	typeTagList       typeTag = 48
	typeTagRuntimeRef typeTag = 49
)
