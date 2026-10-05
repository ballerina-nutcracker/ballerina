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
	help, found, err := signature.At(ctx, sm, request.uri, byteOffsetFromPosition(text, request.params.Position))
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
			Label: label,
			Documentation: protocol.NewOptional(protocol.NewOrParameterInformationDocumentationMarkupContent(
				parameterDocumentationMarkup(doc.Name, doc.Type, doc.Description))),
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
	if help.ActiveParameter != nil {
		result.ActiveParameter = protocol.NewOptionalNullable(protocol.NewOrSignatureHelpActiveParameterUinteger(uint32(*help.ActiveParameter)))
	}
	return trackedResult{handled: true, result: result}
}

func (s *Server) configureSignatureInformation(information protocol.ClientSignatureInformationOptions) {
	if parameters, ok := information.ParameterInformation.Value(); ok {
		s.signatureOffsets, _ = parameters.LabelOffsetSupport.Value()
	}
	if formats, ok := information.DocumentationFormat.Value(); ok && len(formats) > 0 {
		s.signatureMarkdown = formats[0] == protocol.MarkupKindMarkdown
	}
}

func parameterDocumentationMarkup(name, typeText, description string) protocol.MarkupContent {
	value := "**Parameter**  \n**"
	if typeText != "" {
		value += "`" + typeText + "`"
	}
	value += name + "**"
	if description != "" {
		value += ": " + description
	}
	return protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: value}
}
