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

func (*BLangBindingPatternBase) isBindingPattern() {}

type (
	BLangBindingPatternBase struct {
		bLangNodeBase
	}

	BLangCaptureBindingPattern struct {
		BLangBindingPatternBase
		Identifier BLangIdentifier
	}

	BLangErrorBindingPattern struct {
		BLangBindingPatternBase
		ErrorTypeReference         *BLangUserDefinedType
		ErrorMessageBindingPattern *BLangErrorMessageBindingPattern
		ErrorCauseBindingPattern   *BLangErrorCauseBindingPattern
		ErrorFieldBindingPatterns  *BLangErrorFieldBindingPatterns
	}

	BLangErrorMessageBindingPattern struct {
		BLangBindingPatternBase
		SimpleBindingPattern *BLangSimpleBindingPattern
	}
	BLangErrorCauseBindingPattern struct {
		BLangBindingPatternBase
		SimpleBindingPattern *BLangSimpleBindingPattern
		ErrorBindingPattern  *BLangErrorBindingPattern
	}

	BLangErrorFieldBindingPatterns struct {
		BLangBindingPatternBase
		NamedArgBindingPatterns []BLangNamedArgBindingPattern
		RestBindingPattern      *BLangRestBindingPattern
	}
	BLangSimpleBindingPattern struct {
		BLangBindingPatternBase
		CaptureBindingPattern  *BLangCaptureBindingPattern
		WildCardBindingPattern *BLangWildCardBindingPattern
	}

	BLangNamedArgBindingPattern struct {
		BLangBindingPatternBase
		ArgName        *BLangIdentifier
		BindingPattern BindingPatternNode
	}

	BLangRestBindingPattern struct {
		BLangBindingPatternBase
		VariableName *BLangIdentifier
	}

	BLangWildCardBindingPattern struct {
		BLangBindingPatternBase
	}
)

func (*BLangWildCardBindingPattern) isWildCardBindingPattern() {}

var (
	_ CaptureBindingPatternNode      = &BLangCaptureBindingPattern{}
	_ ErrorBindingPatternNode        = &BLangErrorBindingPattern{}
	_ ErrorMessageBindingPatternNode = &BLangErrorMessageBindingPattern{}
	_ ErrorCauseBindingPatternNode   = &BLangErrorCauseBindingPattern{}
	_ ErrorFieldBindingPatternsNode  = &BLangErrorFieldBindingPatterns{}
	_ SimpleBindingPatternNode       = &BLangSimpleBindingPattern{}
	_ NamedArgBindingPatternNode     = &BLangNamedArgBindingPattern{}
	_ RestBindingPatternNode         = &BLangRestBindingPattern{}
	_ WildCardBindingPatternNode     = &BLangWildCardBindingPattern{}
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

func (b *BLangCaptureBindingPattern) SetIdentifier(identifier *BLangIdentifier) {
	b.Identifier = *identifier
}

func (b *BLangErrorBindingPattern) GetErrorTypeReference() UserDefinedTypeNode {
	return b.ErrorTypeReference
}

func (b *BLangErrorBindingPattern) SetErrorTypeReference(userDefinedTypeNode UserDefinedTypeNode) {
	if userDefinedTypeNode, ok := userDefinedTypeNode.(*BLangUserDefinedType); ok {
		b.ErrorTypeReference = userDefinedTypeNode
		return
	}
	panic("userDefinedTypeNode is not a BLangUserDefinedType")
}

func (b *BLangErrorBindingPattern) GetErrorMessageBindingPatternNode() ErrorMessageBindingPatternNode {
	return b.ErrorMessageBindingPattern
}

func (b *BLangErrorBindingPattern) SetErrorMessageBindingPatternNode(errorMessageBindingPatternNode ErrorMessageBindingPatternNode) {
	if errorMessageBindingPatternNode, ok := errorMessageBindingPatternNode.(*BLangErrorMessageBindingPattern); ok {
		b.ErrorMessageBindingPattern = errorMessageBindingPatternNode
		return
	}
	panic("errorMessageBindingPatternNode is not a BLangErrorMessageBindingPattern")
}

func (b *BLangErrorBindingPattern) GetErrorCauseBindingPatternNode() ErrorCauseBindingPatternNode {
	return b.ErrorCauseBindingPattern
}

func (b *BLangErrorBindingPattern) SetErrorCauseBindingPatternNode(errorCauseBindingPatternNode ErrorCauseBindingPatternNode) {
	if errorCauseBindingPatternNode, ok := errorCauseBindingPatternNode.(*BLangErrorCauseBindingPattern); ok {
		b.ErrorCauseBindingPattern = errorCauseBindingPatternNode
		return
	}
	panic("errorCauseBindingPatternNode is not a BLangErrorCauseBindingPattern")
}

func (b *BLangErrorBindingPattern) GetErrorFieldBindingPatternsNode() ErrorFieldBindingPatternsNode {
	return b.ErrorFieldBindingPatterns
}

func (b *BLangErrorBindingPattern) SetErrorFieldBindingPatternsNode(errorFieldBindingPatternsNode ErrorFieldBindingPatternsNode) {
	if errorFieldBindingPatternsNode, ok := errorFieldBindingPatternsNode.(*BLangErrorFieldBindingPatterns); ok {
		b.ErrorFieldBindingPatterns = errorFieldBindingPatternsNode
		return
	}
	panic("errorFieldBindingPatternsNode is not a BLangErrorFieldBindingPatterns")
}

func (b *BLangErrorMessageBindingPattern) GetSimpleBindingPattern() SimpleBindingPatternNode {
	return b.SimpleBindingPattern
}

func (b *BLangErrorMessageBindingPattern) SetSimpleBindingPattern(simpleBindingPattern SimpleBindingPatternNode) {
	if simpleBindingPattern, ok := simpleBindingPattern.(*BLangSimpleBindingPattern); ok {
		b.SimpleBindingPattern = simpleBindingPattern
		return
	}
	panic("simpleBindingPattern is not a BLangSimpleBindingPattern")
}

func (b *BLangErrorCauseBindingPattern) GetSimpleBindingPattern() SimpleBindingPatternNode {
	return b.SimpleBindingPattern
}

func (b *BLangErrorCauseBindingPattern) SetSimpleBindingPattern(simpleBindingPattern SimpleBindingPatternNode) {
	if simpleBindingPattern, ok := simpleBindingPattern.(*BLangSimpleBindingPattern); ok {
		b.SimpleBindingPattern = simpleBindingPattern
		return
	}
	panic("simpleBindingPattern is not a BLangSimpleBindingPattern")
}

func (b *BLangErrorCauseBindingPattern) GetErrorBindingPatternNode() ErrorBindingPatternNode {
	return b.ErrorBindingPattern
}

func (b *BLangErrorCauseBindingPattern) SetErrorBindingPatternNode(errorBindingPatternNode ErrorBindingPatternNode) {
	if errorBindingPatternNode, ok := errorBindingPatternNode.(*BLangErrorBindingPattern); ok {
		b.ErrorBindingPattern = errorBindingPatternNode
		return
	}
	panic("errorBindingPatternNode is not a BLangErrorBindingPattern")
}

func (b *BLangSimpleBindingPattern) GetCaptureBindingPattern() CaptureBindingPatternNode {
	return b.CaptureBindingPattern
}

func (b *BLangSimpleBindingPattern) SetCaptureBindingPattern(captureBindingPattern CaptureBindingPatternNode) {
	if captureBindingPattern, ok := captureBindingPattern.(*BLangCaptureBindingPattern); ok {
		b.CaptureBindingPattern = captureBindingPattern
		return
	}
	panic("captureBindingPattern is not a BLangCaptureBindingPattern")
}

func (b *BLangSimpleBindingPattern) GetWildCardBindingPattern() WildCardBindingPatternNode {
	return b.WildCardBindingPattern
}

func (b *BLangSimpleBindingPattern) SetWildCardBindingPattern(wildCardBindingPattern WildCardBindingPatternNode) {
	if wildCardBindingPatternNode, ok := wildCardBindingPattern.(*BLangWildCardBindingPattern); ok {
		b.WildCardBindingPattern = wildCardBindingPatternNode
		return
	}
	panic("wildCardBindingPatternNode is not a BLangWildCardBindingPattern")
}

func (b *BLangErrorFieldBindingPatterns) GetNamedArgMatchPatterns() []NamedArgBindingPatternNode {
	namedArgBindingPatterns := make([]NamedArgBindingPatternNode, len(b.NamedArgBindingPatterns))
	for i := range b.NamedArgBindingPatterns {
		namedArgBindingPatterns[i] = &b.NamedArgBindingPatterns[i]
	}
	return namedArgBindingPatterns
}

func (b *BLangErrorFieldBindingPatterns) AddNamedArgBindingPattern(namedArgBindingPatternNode NamedArgBindingPatternNode) {
	if namedArgBindingPatternNode, ok := namedArgBindingPatternNode.(*BLangNamedArgBindingPattern); ok {
		b.NamedArgBindingPatterns = append(b.NamedArgBindingPatterns, *namedArgBindingPatternNode)
		return
	}
	panic("namedArgBindingPatternNode is not a BLangNamedArgBindingPattern")
}

func (b *BLangErrorFieldBindingPatterns) GetRestBindingPattern() RestBindingPatternNode {
	return b.RestBindingPattern
}

func (b *BLangErrorFieldBindingPatterns) SetRestBindingPattern(restBindingPattern RestBindingPatternNode) {
	if restBindingPattern, ok := restBindingPattern.(*BLangRestBindingPattern); ok {
		b.RestBindingPattern = restBindingPattern
		return
	}
	panic("restBindingPattern is not a BLangRestBindingPattern")
}

func (b *BLangNamedArgBindingPattern) GetIdentifier() *BLangIdentifier {
	return b.ArgName
}

func (b *BLangNamedArgBindingPattern) SetIdentifier(variableName *BLangIdentifier) {
	b.ArgName = variableName
}

func (b *BLangNamedArgBindingPattern) GetBindingPattern() BindingPatternNode {
	return b.BindingPattern
}

func (b *BLangNamedArgBindingPattern) SetBindingPattern(bindingPattern BindingPatternNode) {
	b.BindingPattern = bindingPattern
}

func (b *BLangRestBindingPattern) GetIdentifier() *BLangIdentifier {
	return b.VariableName
}

func (b *BLangRestBindingPattern) SetIdentifier(variableName *BLangIdentifier) {
	b.VariableName = variableName
}

func (*BLangWildCardBindingPattern) actionOrExpression() {}
func (*BLangWildCardBindingPattern) expressionNode()     {}
func (*BLangWildCardBindingPattern) isLExpr()            {}
