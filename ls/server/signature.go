// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"encoding/json"
	"unicode/utf16"

	"github.com/ballerina-nutcracker/ballerina/ls/core/signature"
	"github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
	"github.com/ballerina-nutcracker/ballerina/ls/protocol"
)

type signatureRequest struct {
	params    protocol.SignatureHelpParams
	uri       workspace.DocumentURI
	text      string
	available bool
}

func (s *Server) captureSignature(params json.RawMessage) *signatureRequest {
	request := &signatureRequest{}
	if json.Unmarshal(params, &request.params) != nil {
		return request
	}
	uri, err := workspace.NewFileURI(request.params.TextDocument.URI)
	if err != nil {
		return request
	}
	request.uri = uri
	request.text, request.available = s.projects.DocumentText(uri)
	return request
}

func (s *Server) handleSignature(ctx context.Context, request *signatureRequest) trackedResult {
	empty := trackedResult{handled: true}
	if !request.available {
		return empty
	}
	sm, ok := s.compiler.SealedModuleFor(ctx, request.uri)
	if ctx.Err() != nil {
		return trackedResult{handled: true, err: &protocol.RPCError{Code: rpcRequestCancelled, Message: "request cancelled"}}
	}
	if !ok {
		return empty
	}
	text, ok := sm.SourceText(request.uri)
	if !ok || text != request.text {
		return empty
	}
	current, ok := s.projects.DocumentText(request.uri)
	if !ok || current != request.text {
		return empty
	}
	offset, valid := utf16Offset(text, request.params.Position)
	if !valid {
		return trackedResult{handled: true, err: &protocol.RPCError{Code: rpcInvalidParams, Message: "invalid signature help position"}}
	}
	help, found, err := signature.At(ctx, sm, request.uri, offset)
	if err != nil {
		if ctx.Err() != nil {
			return trackedResult{handled: true, err: &protocol.RPCError{Code: rpcRequestCancelled, Message: "request cancelled"}}
		}
		return trackedResult{handled: true, err: &protocol.RPCError{Code: rpcInternalError, Message: "signature help failed"}}
	}
	if !found {
		return empty
	}
	current, ok = s.projects.DocumentText(request.uri)
	if !ok || current != request.text {
		return empty
	}
	parameters := make([]protocol.ParameterInformation, 0, len(help.Parameters))
	for i, span := range help.Parameters {
		label := protocol.NewOrParameterInformationLabelString(help.Label[span[0]:span[1]])
		if s.signatureOffsets {
			label = protocol.NewOrParameterInformationLabelVariant1(protocol.TupleParameterInformationLabelItem1{
				Item0: uint32(len(utf16.Encode([]rune(help.Label[:span[0]])))),
				Item1: uint32(len(utf16.Encode([]rune(help.Label[:span[1]])))),
			})
		}
		doc := help.ParameterDocumentation[i]
		parameters = append(parameters, protocol.ParameterInformation{
			Label:         label,
			Documentation: protocol.NewOptional(s.parameterDocumentation(doc.Name, doc.Type, doc.Description)),
		})
	}
	result := protocol.SignatureHelp{Signatures: []protocol.SignatureInformation{{Label: help.Label, Parameters: protocol.NewOptional(parameters)}}, ActiveSignature: protocol.NewOptional(uint32(0))}
	if help.Description != "" {
		documentation := protocol.NewOrSignatureInformationDocumentationString("Description\n" + help.Description)
		if s.signatureMarkdown {
			documentation = protocol.NewOrSignatureInformationDocumentationMarkupContent(protocol.MarkupContent{
				Kind: protocol.MarkupKindMarkdown, Value: "**Description**  \n" + help.Description,
			})
		}
		result.Signatures[0].Documentation = protocol.NewOptional(documentation)
	}
	result.ActiveParameter = s.activeParameter(help.ActiveParameter, len(parameters))
	return trackedResult{handled: true, result: result}
}

// activeParameter encodes a missing active parameter as null when the client supports it.
// Otherwise it uses an out-of-range index, because an omitted value defaults to 0.
func (s *Server) activeParameter(active *int, parameters int) protocol.OptionalNullable[protocol.OrSignatureHelpActiveParameter] {
	switch {
	case active != nil:
		return protocol.NewOptionalNullable(protocol.NewOrSignatureHelpActiveParameterUinteger(uint32(*active)))
	case parameters == 0:
		return protocol.OptionalNullable[protocol.OrSignatureHelpActiveParameter]{}
	case s.signatureNoActive:
		return protocol.NullOptionalNullable[protocol.OrSignatureHelpActiveParameter]()
	default:
		return protocol.NewOptionalNullable(protocol.NewOrSignatureHelpActiveParameterUinteger(uint32(parameters)))
	}
}

func (s *Server) configureSignatureInformation(information protocol.ClientSignatureInformationOptions) {
	s.signatureNoActive, _ = information.NoActiveParameterSupport.Value()
	if parameters, ok := information.ParameterInformation.Value(); ok {
		s.signatureOffsets, _ = parameters.LabelOffsetSupport.Value()
	}
	if formats, ok := information.DocumentationFormat.Value(); ok && len(formats) > 0 {
		s.signatureMarkdown = formats[0] == protocol.MarkupKindMarkdown
	}
}

func (s *Server) parameterDocumentation(name, typeText, description string) protocol.OrParameterInformationDocumentation {
	if s.signatureMarkdown {
		return protocol.NewOrParameterInformationDocumentationMarkupContent(parameterDocumentationMarkup(name, typeText, description))
	}
	return protocol.NewOrParameterInformationDocumentationString("Parameter\n" + parameterDocumentationText(name, typeText, description))
}

func parameterDocumentationMarkup(name, typeText, description string) protocol.MarkupContent {
	value := "**Parameter**  \n**"
	if typeText != "" {
		value += "`" + typeText + "`"
		if name != "" {
			value += " "
		}
	}
	value += name + "**"
	if description != "" {
		value += ": " + description
	}
	return protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: value}
}

func parameterDocumentationText(name, typeText, description string) string {
	value := typeText
	if typeText != "" && name != "" {
		value += " "
	}
	value += name
	if description != "" {
		value += ": " + description
	}
	return value
}
