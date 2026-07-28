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

package ast

func (*bLangBindingPatternBase) isBindingPattern() {}

type (
	bLangBindingPatternBase struct {
		bLangNodeBase
	}

	BLangCaptureBindingPattern struct {
		bLangBindingPatternBase
		Identifier BLangIdentifier
	}

	BLangErrorBindingPattern struct {
		bLangBindingPatternBase
		ErrorTypeReference         *BLangUserDefinedType
		ErrorMessageBindingPattern *BLangErrorMessageBindingPattern
		ErrorCauseBindingPattern   *BLangErrorCauseBindingPattern
		ErrorFieldBindingPatterns  *BLangErrorFieldBindingPatterns
	}

	BLangErrorMessageBindingPattern struct {
		bLangBindingPatternBase
		SimpleBindingPattern *BLangSimpleBindingPattern
	}
	BLangErrorCauseBindingPattern struct {
		bLangBindingPatternBase
		SimpleBindingPattern *BLangSimpleBindingPattern
		ErrorBindingPattern  *BLangErrorBindingPattern
	}

	BLangErrorFieldBindingPatterns struct {
		bLangBindingPatternBase
		NamedArgBindingPatterns []BLangNamedArgBindingPattern
		RestBindingPattern      *BLangRestBindingPattern
	}
	BLangSimpleBindingPattern struct {
		bLangBindingPatternBase
		CaptureBindingPattern  *BLangCaptureBindingPattern
		WildCardBindingPattern *BLangWildCardBindingPattern
	}

	BLangNamedArgBindingPattern struct {
		bLangBindingPatternBase
		ArgName        *BLangIdentifier
		BindingPattern BindingPatternNode
	}

	BLangRestBindingPattern struct {
		bLangBindingPatternBase
		VariableName *BLangIdentifier
	}

	BLangWildCardBindingPattern struct {
		bLangBindingPatternBase
	}
)

func (*BLangWildCardBindingPattern) isWildCardBindingPattern() {}

var (
	_ CaptureBindingPatternNode      = &BLangCaptureBindingPattern{}
	_ ErrorMessageBindingPatternNode = &BLangErrorMessageBindingPattern{}
	_ RestBindingPatternNode         = &BLangRestBindingPattern{}
)

var (
	_ BLangNode = &BLangCaptureBindingPattern{}
	_ BLangNode = &BLangErrorBindingPattern{}
	_ BLangNode = &BLangErrorMessageBindingPattern{}
	_ BLangNode = &BLangErrorCauseBindingPattern{}
	_ BLangNode = &BLangErrorFieldBindingPatterns{}
	_ BLangNode = &BLangSimpleBindingPattern{}
	_ BLangNode = &BLangNamedArgBindingPattern{}
	_ BLangNode = &BLangRestBindingPattern{}
	_ BLangNode = &BLangWildCardBindingPattern{}
)

func (b *BLangCaptureBindingPattern) GetIdentifier() *BLangIdentifier {
	return &b.Identifier
}

func (b *BLangErrorBindingPattern) GetErrorTypeReference() *BLangUserDefinedType {
	return b.ErrorTypeReference
}

func (b *BLangErrorBindingPattern) GetErrorMessageBindingPatternNode() ErrorMessageBindingPatternNode {
	return b.ErrorMessageBindingPattern
}

func (b *BLangErrorBindingPattern) GetErrorCauseBindingPatternNode() *BLangErrorCauseBindingPattern {
	return b.ErrorCauseBindingPattern
}

func (b *BLangErrorBindingPattern) GetErrorFieldBindingPatternsNode() *BLangErrorFieldBindingPatterns {
	return b.ErrorFieldBindingPatterns
}

func (b *BLangErrorMessageBindingPattern) GetSimpleBindingPattern() *BLangSimpleBindingPattern {
	return b.SimpleBindingPattern
}

func (b *BLangErrorCauseBindingPattern) GetSimpleBindingPattern() *BLangSimpleBindingPattern {
	return b.SimpleBindingPattern
}

func (b *BLangErrorCauseBindingPattern) GetErrorBindingPatternNode() *BLangErrorBindingPattern {
	return b.ErrorBindingPattern
}

func (b *BLangSimpleBindingPattern) GetCaptureBindingPattern() CaptureBindingPatternNode {
	return b.CaptureBindingPattern
}

func (b *BLangSimpleBindingPattern) GetWildCardBindingPattern() *BLangWildCardBindingPattern {
	return b.WildCardBindingPattern
}

func (b *BLangErrorFieldBindingPatterns) GetNamedArgMatchPatterns() []*BLangNamedArgBindingPattern {
	namedArgBindingPatterns := make([]*BLangNamedArgBindingPattern, len(b.NamedArgBindingPatterns))
	for i := range b.NamedArgBindingPatterns {
		namedArgBindingPatterns[i] = &b.NamedArgBindingPatterns[i]
	}
	return namedArgBindingPatterns
}

func (b *BLangErrorFieldBindingPatterns) AddNamedArgBindingPattern(namedArgBindingPatternNode *BLangNamedArgBindingPattern) {
	b.NamedArgBindingPatterns = append(b.NamedArgBindingPatterns, *namedArgBindingPatternNode)
}

func (b *BLangErrorFieldBindingPatterns) GetRestBindingPattern() RestBindingPatternNode {
	return b.RestBindingPattern
}

func (b *BLangNamedArgBindingPattern) GetIdentifier() *BLangIdentifier {
	return b.ArgName
}

func (b *BLangNamedArgBindingPattern) GetBindingPattern() BindingPatternNode {
	return b.BindingPattern
}

func (b *BLangRestBindingPattern) GetIdentifier() *BLangIdentifier {
	return b.VariableName
}

func (*BLangWildCardBindingPattern) actionOrExpression() {}
func (*BLangWildCardBindingPattern) expressionNode()     {}
func (*BLangWildCardBindingPattern) isLExpr()            {}
