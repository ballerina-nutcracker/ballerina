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
// KIND, either express or implied. See the License for the
// specific language governing permissions and limitations
// under the License.

// Code generated from the Language Server Protocol 3.18 metamodel.
// DO NOT EDIT.

package protocol

import (
	"encoding/json"
	"fmt"
)

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if o.value == nil {
		return nil, fmt.Errorf("optional field is not set")
	}
	return json.Marshal(*o.value)
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return fmt.Errorf("optional field cannot be null or empty")
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = Optional[T]{value: &v}
	return nil
}

func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	if n.null {
		return []byte("null"), nil
	}
	if n.value == nil {
		return nil, fmt.Errorf("nullable field has no value and is not null")
	}
	return json.Marshal(*n.value)
}

func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("nullable field cannot be empty")
	}
	if string(data) == "null" {
		*n = Nullable[T]{null: true}
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*n = NewNullable(v)
	return nil
}

func (o OptionalNullable[T]) MarshalJSON() ([]byte, error) {
	if !o.IsSet() {
		return nil, fmt.Errorf("optional nullable field is not set")
	}
	if o.null {
		return []byte("null"), nil
	}
	if o.value == nil {
		return nil, fmt.Errorf("optional nullable field is set without value or null")
	}
	return json.Marshal(*o.value)
}

func (o *OptionalNullable[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("optional nullable field cannot be empty")
	}
	if string(data) == "null" {
		*o = OptionalNullable[T]{null: true}
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = NewOptionalNullable(v)
	return nil
}

func isString(raw any) bool {
	_, ok := raw.(string)
	return ok
}

func isNumber(raw any) bool {
	_, ok := raw.(float64)
	return ok
}

func isBool(raw any) bool {
	_, ok := raw.(bool)
	return ok
}

func isObject(raw any) bool {
	_, ok := raw.(map[string]any)
	return ok
}

func isArray(raw any) bool {
	_, ok := raw.([]any)
	return ok
}

func hasKey(raw any, key string) bool {
	obj, ok := raw.(map[string]any)
	if !ok || obj == nil {
		return false
	}
	_, has := obj[key]
	return has
}

func isStringLiteral(raw any, want string) bool {
	s, ok := raw.(string)
	return ok && s == want
}

func isIntegerLiteral(raw any, want int64) bool {
	n, ok := raw.(float64)
	return ok && int64(n) == want
}

func annotatedTextEditMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["annotationId"]; !has {
		return false
	}
	if _, has := obj["newText"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func applyWorkspaceEditParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["edit"]; !has {
		return false
	}
	return true
}

func applyWorkspaceEditResultMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["applied"]; !has {
		return false
	}
	return true
}

func baseSymbolInformationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["name"]; !has {
		return false
	}
	return true
}

func callHierarchyClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func callHierarchyIncomingCallMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["from"]; !has {
		return false
	}
	if _, has := obj["fromRanges"]; !has {
		return false
	}
	return true
}

func callHierarchyIncomingCallsParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["item"]; !has {
		return false
	}
	return true
}

func callHierarchyItemMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["name"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["selectionRange"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func callHierarchyOptionsMatches(raw any) bool {
	return isObject(raw)
}

func callHierarchyOutgoingCallMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["fromRanges"]; !has {
		return false
	}
	if _, has := obj["to"]; !has {
		return false
	}
	return true
}

func callHierarchyOutgoingCallsParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["item"]; !has {
		return false
	}
	return true
}

func callHierarchyPrepareParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func callHierarchyRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func cancelParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["id"]; !has {
		return false
	}
	return true
}

func changeAnnotationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["label"]; !has {
		return false
	}
	return true
}

func changeAnnotationsSupportOptionsMatches(raw any) bool {
	return isObject(raw)
}

func clientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func clientCodeActionKindOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["valueSet"]; !has {
		return false
	}
	return true
}

func clientCodeActionLiteralOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["codeActionKind"]; !has {
		return false
	}
	return true
}

func clientCodeActionResolveOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["properties"]; !has {
		return false
	}
	return true
}

func clientCodeLensResolveOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["properties"]; !has {
		return false
	}
	return true
}

func clientCompletionItemInsertTextModeOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["valueSet"]; !has {
		return false
	}
	return true
}

func clientCompletionItemOptionsMatches(raw any) bool {
	return isObject(raw)
}

func clientCompletionItemOptionsKindMatches(raw any) bool {
	return isObject(raw)
}

func clientCompletionItemResolveOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["properties"]; !has {
		return false
	}
	return true
}

func clientDiagnosticsTagOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["valueSet"]; !has {
		return false
	}
	return true
}

func clientFoldingRangeKindOptionsMatches(raw any) bool {
	return isObject(raw)
}

func clientFoldingRangeOptionsMatches(raw any) bool {
	return isObject(raw)
}

func clientInfoMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["name"]; !has {
		return false
	}
	return true
}

func clientInlayHintResolveOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["properties"]; !has {
		return false
	}
	return true
}

func clientSemanticTokensRequestFullDeltaMatches(raw any) bool {
	return isObject(raw)
}

func clientSemanticTokensRequestOptionsMatches(raw any) bool {
	return isObject(raw)
}

func clientShowMessageActionItemOptionsMatches(raw any) bool {
	return isObject(raw)
}

func clientSignatureInformationOptionsMatches(raw any) bool {
	return isObject(raw)
}

func clientSignatureParameterInformationOptionsMatches(raw any) bool {
	return isObject(raw)
}

func clientSymbolKindOptionsMatches(raw any) bool {
	return isObject(raw)
}

func clientSymbolResolveOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["properties"]; !has {
		return false
	}
	return true
}

func clientSymbolTagOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["valueSet"]; !has {
		return false
	}
	return true
}

func codeActionMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["title"]; !has {
		return false
	}
	return true
}

func codeActionClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func codeActionContextMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["diagnostics"]; !has {
		return false
	}
	return true
}

func codeActionDisabledMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["reason"]; !has {
		return false
	}
	return true
}

func codeActionKindDocumentationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["command"]; !has {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	return true
}

func codeActionOptionsMatches(raw any) bool {
	return isObject(raw)
}

func codeActionParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["context"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func codeActionRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func codeActionTagOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["valueSet"]; !has {
		return false
	}
	return true
}

func codeDescriptionMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["href"]; !has {
		return false
	}
	return true
}

func codeLensMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func codeLensClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func codeLensOptionsMatches(raw any) bool {
	return isObject(raw)
}

func codeLensParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func codeLensRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func codeLensWorkspaceClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func colorMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["alpha"]; !has {
		return false
	}
	if _, has := obj["blue"]; !has {
		return false
	}
	if _, has := obj["green"]; !has {
		return false
	}
	if _, has := obj["red"]; !has {
		return false
	}
	return true
}

func colorInformationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["color"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func colorPresentationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["label"]; !has {
		return false
	}
	return true
}

func colorPresentationParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["color"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func commandMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["command"]; !has {
		return false
	}
	if _, has := obj["title"]; !has {
		return false
	}
	return true
}

func completionClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func completionContextMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["triggerKind"]; !has {
		return false
	}
	return true
}

func completionItemMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["label"]; !has {
		return false
	}
	return true
}

func completionItemApplyKindsMatches(raw any) bool {
	return isObject(raw)
}

func completionItemDefaultsMatches(raw any) bool {
	return isObject(raw)
}

func completionItemLabelDetailsMatches(raw any) bool {
	return isObject(raw)
}

func completionItemTagOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["valueSet"]; !has {
		return false
	}
	return true
}

func completionListMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["isIncomplete"]; !has {
		return false
	}
	if _, has := obj["items"]; !has {
		return false
	}
	return true
}

func completionListCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func completionOptionsMatches(raw any) bool {
	return isObject(raw)
}

func completionParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func completionRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func configurationItemMatches(raw any) bool {
	return isObject(raw)
}

func configurationParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["items"]; !has {
		return false
	}
	return true
}

func createFileMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func createFileOptionsMatches(raw any) bool {
	return isObject(raw)
}

func createFilesParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["files"]; !has {
		return false
	}
	return true
}

func declarationClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func declarationOptionsMatches(raw any) bool {
	return isObject(raw)
}

func declarationParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func declarationRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func definitionClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func definitionOptionsMatches(raw any) bool {
	return isObject(raw)
}

func definitionParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func definitionRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func deleteFileMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func deleteFileOptionsMatches(raw any) bool {
	return isObject(raw)
}

func deleteFilesParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["files"]; !has {
		return false
	}
	return true
}

func diagnosticMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["message"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func diagnosticClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func diagnosticOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["interFileDependencies"]; !has {
		return false
	}
	if _, has := obj["workspaceDiagnostics"]; !has {
		return false
	}
	return true
}

func diagnosticRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	if _, has := obj["interFileDependencies"]; !has {
		return false
	}
	if _, has := obj["workspaceDiagnostics"]; !has {
		return false
	}
	return true
}

func diagnosticRelatedInformationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["location"]; !has {
		return false
	}
	if _, has := obj["message"]; !has {
		return false
	}
	return true
}

func diagnosticServerCancellationDataMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["retriggerRequest"]; !has {
		return false
	}
	return true
}

func diagnosticWorkspaceClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func diagnosticsCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func didChangeConfigurationClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func didChangeConfigurationParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["settings"]; !has {
		return false
	}
	return true
}

func didChangeConfigurationRegistrationOptionsMatches(raw any) bool {
	return isObject(raw)
}

func didChangeNotebookDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["change"]; !has {
		return false
	}
	if _, has := obj["notebookDocument"]; !has {
		return false
	}
	return true
}

func didChangeTextDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["contentChanges"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func didChangeWatchedFilesClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func didChangeWatchedFilesParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["changes"]; !has {
		return false
	}
	return true
}

func didChangeWatchedFilesRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["watchers"]; !has {
		return false
	}
	return true
}

func didChangeWorkspaceFoldersParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["event"]; !has {
		return false
	}
	return true
}

func didCloseNotebookDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["cellTextDocuments"]; !has {
		return false
	}
	if _, has := obj["notebookDocument"]; !has {
		return false
	}
	return true
}

func didCloseTextDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func didOpenNotebookDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["cellTextDocuments"]; !has {
		return false
	}
	if _, has := obj["notebookDocument"]; !has {
		return false
	}
	return true
}

func didOpenTextDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func didSaveNotebookDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["notebookDocument"]; !has {
		return false
	}
	return true
}

func didSaveTextDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentColorClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func documentColorOptionsMatches(raw any) bool {
	return isObject(raw)
}

func documentColorParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentColorRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func documentDiagnosticParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentDiagnosticReportPartialResultMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["relatedDocuments"]; !has {
		return false
	}
	return true
}

func documentFormattingClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func documentFormattingOptionsMatches(raw any) bool {
	return isObject(raw)
}

func documentFormattingParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["options"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentFormattingRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func documentHighlightMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func documentHighlightClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func documentHighlightOptionsMatches(raw any) bool {
	return isObject(raw)
}

func documentHighlightParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentHighlightRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func documentLinkMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func documentLinkClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func documentLinkOptionsMatches(raw any) bool {
	return isObject(raw)
}

func documentLinkParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentLinkRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func documentOnTypeFormattingClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func documentOnTypeFormattingOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["firstTriggerCharacter"]; !has {
		return false
	}
	return true
}

func documentOnTypeFormattingParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["ch"]; !has {
		return false
	}
	if _, has := obj["options"]; !has {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentOnTypeFormattingRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	if _, has := obj["firstTriggerCharacter"]; !has {
		return false
	}
	return true
}

func documentRangeFormattingClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func documentRangeFormattingOptionsMatches(raw any) bool {
	return isObject(raw)
}

func documentRangeFormattingParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["options"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentRangeFormattingRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func documentRangesFormattingParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["options"]; !has {
		return false
	}
	if _, has := obj["ranges"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentSymbolMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["name"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["selectionRange"]; !has {
		return false
	}
	return true
}

func documentSymbolClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func documentSymbolOptionsMatches(raw any) bool {
	return isObject(raw)
}

func documentSymbolParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func documentSymbolRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func editRangeWithInsertReplaceMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["insert"]; !has {
		return false
	}
	if _, has := obj["replace"]; !has {
		return false
	}
	return true
}

func executeCommandClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func executeCommandOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["commands"]; !has {
		return false
	}
	return true
}

func executeCommandParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["command"]; !has {
		return false
	}
	return true
}

func executeCommandRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["commands"]; !has {
		return false
	}
	return true
}

func executionSummaryMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["executionOrder"]; !has {
		return false
	}
	return true
}

func fileCreateMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func fileDeleteMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func fileEventMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["type"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func fileOperationClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func fileOperationFilterMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["pattern"]; !has {
		return false
	}
	return true
}

func fileOperationOptionsMatches(raw any) bool {
	return isObject(raw)
}

func fileOperationPatternMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["glob"]; !has {
		return false
	}
	return true
}

func fileOperationPatternOptionsMatches(raw any) bool {
	return isObject(raw)
}

func fileOperationRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["filters"]; !has {
		return false
	}
	return true
}

func fileRenameMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["newUri"]; !has {
		return false
	}
	if _, has := obj["oldUri"]; !has {
		return false
	}
	return true
}

func fileSystemWatcherMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["globPattern"]; !has {
		return false
	}
	return true
}

func foldingRangeMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["endLine"]; !has {
		return false
	}
	if _, has := obj["startLine"]; !has {
		return false
	}
	return true
}

func foldingRangeClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func foldingRangeOptionsMatches(raw any) bool {
	return isObject(raw)
}

func foldingRangeParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func foldingRangeRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func foldingRangeWorkspaceClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func formattingOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["insertSpaces"]; !has {
		return false
	}
	if _, has := obj["tabSize"]; !has {
		return false
	}
	return true
}

func fullDocumentDiagnosticReportMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["items"]; !has {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	return true
}

func generalClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func hoverMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["contents"]; !has {
		return false
	}
	return true
}

func hoverClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func hoverOptionsMatches(raw any) bool {
	return isObject(raw)
}

func hoverParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func hoverRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func implementationClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func implementationOptionsMatches(raw any) bool {
	return isObject(raw)
}

func implementationParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func implementationRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func initializeErrorMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["retry"]; !has {
		return false
	}
	return true
}

func initializeParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["capabilities"]; !has {
		return false
	}
	if _, has := obj["processId"]; !has {
		return false
	}
	if _, has := obj["rootUri"]; !has {
		return false
	}
	return true
}

func initializeResultMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["capabilities"]; !has {
		return false
	}
	return true
}

func initializedParamsMatches(raw any) bool {
	return isObject(raw)
}

func inlayHintMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["label"]; !has {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	return true
}

func inlayHintClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func inlayHintLabelPartMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["value"]; !has {
		return false
	}
	return true
}

func inlayHintOptionsMatches(raw any) bool {
	return isObject(raw)
}

func inlayHintParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func inlayHintRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func inlayHintWorkspaceClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func inlineCompletionClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func inlineCompletionContextMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["triggerKind"]; !has {
		return false
	}
	return true
}

func inlineCompletionItemMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["insertText"]; !has {
		return false
	}
	return true
}

func inlineCompletionListMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["items"]; !has {
		return false
	}
	return true
}

func inlineCompletionOptionsMatches(raw any) bool {
	return isObject(raw)
}

func inlineCompletionParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["context"]; !has {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func inlineCompletionRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func inlineValueClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func inlineValueContextMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["frameId"]; !has {
		return false
	}
	if _, has := obj["stoppedLocation"]; !has {
		return false
	}
	return true
}

func inlineValueEvaluatableExpressionMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func inlineValueOptionsMatches(raw any) bool {
	return isObject(raw)
}

func inlineValueParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["context"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func inlineValueRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func inlineValueTextMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["text"]; !has {
		return false
	}
	return true
}

func inlineValueVariableLookupMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["caseSensitiveLookup"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func inlineValueWorkspaceClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func insertReplaceEditMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["insert"]; !has {
		return false
	}
	if _, has := obj["newText"]; !has {
		return false
	}
	if _, has := obj["replace"]; !has {
		return false
	}
	return true
}

func linkedEditingRangeClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func linkedEditingRangeOptionsMatches(raw any) bool {
	return isObject(raw)
}

func linkedEditingRangeParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func linkedEditingRangeRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func linkedEditingRangesMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["ranges"]; !has {
		return false
	}
	return true
}

func locationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func locationLinkMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["targetRange"]; !has {
		return false
	}
	if _, has := obj["targetSelectionRange"]; !has {
		return false
	}
	if _, has := obj["targetUri"]; !has {
		return false
	}
	return true
}

func locationUriOnlyMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func logMessageParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["message"]; !has {
		return false
	}
	if _, has := obj["type"]; !has {
		return false
	}
	return true
}

func logTraceParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["message"]; !has {
		return false
	}
	return true
}

func markdownClientCapabilitiesMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["parser"]; !has {
		return false
	}
	return true
}

func markedStringWithLanguageMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["language"]; !has {
		return false
	}
	if _, has := obj["value"]; !has {
		return false
	}
	return true
}

func markupContentMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["value"]; !has {
		return false
	}
	return true
}

func messageActionItemMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["title"]; !has {
		return false
	}
	return true
}

func monikerMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["identifier"]; !has {
		return false
	}
	if _, has := obj["scheme"]; !has {
		return false
	}
	if _, has := obj["unique"]; !has {
		return false
	}
	return true
}

func monikerClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func monikerOptionsMatches(raw any) bool {
	return isObject(raw)
}

func monikerParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func monikerRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func notebookCellMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["document"]; !has {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	return true
}

func notebookCellArrayChangeMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["deleteCount"]; !has {
		return false
	}
	if _, has := obj["start"]; !has {
		return false
	}
	return true
}

func notebookCellLanguageMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["language"]; !has {
		return false
	}
	return true
}

func notebookCellTextDocumentFilterMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["notebook"]; !has {
		return false
	}
	return true
}

func notebookDocumentMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["cells"]; !has {
		return false
	}
	if _, has := obj["notebookType"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	if _, has := obj["version"]; !has {
		return false
	}
	return true
}

func notebookDocumentCellChangeStructureMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["array"]; !has {
		return false
	}
	return true
}

func notebookDocumentCellChangesMatches(raw any) bool {
	return isObject(raw)
}

func notebookDocumentCellContentChangesMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["changes"]; !has {
		return false
	}
	if _, has := obj["document"]; !has {
		return false
	}
	return true
}

func notebookDocumentChangeEventMatches(raw any) bool {
	return isObject(raw)
}

func notebookDocumentClientCapabilitiesMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["synchronization"]; !has {
		return false
	}
	return true
}

func notebookDocumentFilterNotebookTypeMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["notebookType"]; !has {
		return false
	}
	return true
}

func notebookDocumentFilterPatternMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["pattern"]; !has {
		return false
	}
	return true
}

func notebookDocumentFilterSchemeMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["scheme"]; !has {
		return false
	}
	return true
}

func notebookDocumentFilterWithCellsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["cells"]; !has {
		return false
	}
	return true
}

func notebookDocumentFilterWithNotebookMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["notebook"]; !has {
		return false
	}
	return true
}

func notebookDocumentIdentifierMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func notebookDocumentSyncClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func notebookDocumentSyncOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["notebookSelector"]; !has {
		return false
	}
	return true
}

func notebookDocumentSyncRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["notebookSelector"]; !has {
		return false
	}
	return true
}

func optionalVersionedTextDocumentIdentifierMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	if _, has := obj["version"]; !has {
		return false
	}
	return true
}

func parameterInformationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["label"]; !has {
		return false
	}
	return true
}

func partialResultParamsMatches(raw any) bool {
	return isObject(raw)
}

func positionMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["character"]; !has {
		return false
	}
	if _, has := obj["line"]; !has {
		return false
	}
	return true
}

func prepareRenameDefaultBehaviorMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["defaultBehavior"]; !has {
		return false
	}
	return true
}

func prepareRenameParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func prepareRenamePlaceholderMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["placeholder"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func previousResultIDMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	if _, has := obj["value"]; !has {
		return false
	}
	return true
}

func progressParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["token"]; !has {
		return false
	}
	if _, has := obj["value"]; !has {
		return false
	}
	return true
}

func publishDiagnosticsClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func publishDiagnosticsParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["diagnostics"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func rangeMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["end"]; !has {
		return false
	}
	if _, has := obj["start"]; !has {
		return false
	}
	return true
}

func referenceClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func referenceContextMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["includeDeclaration"]; !has {
		return false
	}
	return true
}

func referenceOptionsMatches(raw any) bool {
	return isObject(raw)
}

func referenceParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["context"]; !has {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func referenceRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func registrationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["id"]; !has {
		return false
	}
	if _, has := obj["method"]; !has {
		return false
	}
	return true
}

func registrationParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["registrations"]; !has {
		return false
	}
	return true
}

func regularExpressionsClientCapabilitiesMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["engine"]; !has {
		return false
	}
	return true
}

func relatedFullDocumentDiagnosticReportMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["items"]; !has {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	return true
}

func relatedUnchangedDocumentDiagnosticReportMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["resultId"]; !has {
		return false
	}
	return true
}

func relativePatternMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["baseUri"]; !has {
		return false
	}
	if _, has := obj["pattern"]; !has {
		return false
	}
	return true
}

func renameClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func renameFileMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["newUri"]; !has {
		return false
	}
	if _, has := obj["oldUri"]; !has {
		return false
	}
	return true
}

func renameFileOptionsMatches(raw any) bool {
	return isObject(raw)
}

func renameFilesParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["files"]; !has {
		return false
	}
	return true
}

func renameOptionsMatches(raw any) bool {
	return isObject(raw)
}

func renameParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["newName"]; !has {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func renameRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func resourceOperationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	return true
}

func saveOptionsMatches(raw any) bool {
	return isObject(raw)
}

func selectedCompletionInfoMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["text"]; !has {
		return false
	}
	return true
}

func selectionRangeMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func selectionRangeClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func selectionRangeOptionsMatches(raw any) bool {
	return isObject(raw)
}

func selectionRangeParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["positions"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func selectionRangeRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func semanticTokensMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["data"]; !has {
		return false
	}
	return true
}

func semanticTokensClientCapabilitiesMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["formats"]; !has {
		return false
	}
	if _, has := obj["requests"]; !has {
		return false
	}
	if _, has := obj["tokenModifiers"]; !has {
		return false
	}
	if _, has := obj["tokenTypes"]; !has {
		return false
	}
	return true
}

func semanticTokensDeltaMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["edits"]; !has {
		return false
	}
	return true
}

func semanticTokensDeltaParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["previousResultId"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func semanticTokensDeltaPartialResultMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["edits"]; !has {
		return false
	}
	return true
}

func semanticTokensEditMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["deleteCount"]; !has {
		return false
	}
	if _, has := obj["start"]; !has {
		return false
	}
	return true
}

func semanticTokensFullDeltaMatches(raw any) bool {
	return isObject(raw)
}

func semanticTokensLegendMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["tokenModifiers"]; !has {
		return false
	}
	if _, has := obj["tokenTypes"]; !has {
		return false
	}
	return true
}

func semanticTokensOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["legend"]; !has {
		return false
	}
	return true
}

func semanticTokensParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func semanticTokensPartialResultMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["data"]; !has {
		return false
	}
	return true
}

func semanticTokensRangeParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func semanticTokensRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	if _, has := obj["legend"]; !has {
		return false
	}
	return true
}

func semanticTokensWorkspaceClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func serverCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func serverCompletionItemOptionsMatches(raw any) bool {
	return isObject(raw)
}

func serverInfoMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["name"]; !has {
		return false
	}
	return true
}

func setTraceParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["value"]; !has {
		return false
	}
	return true
}

func showDocumentClientCapabilitiesMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["support"]; !has {
		return false
	}
	return true
}

func showDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func showDocumentResultMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["success"]; !has {
		return false
	}
	return true
}

func showMessageParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["message"]; !has {
		return false
	}
	if _, has := obj["type"]; !has {
		return false
	}
	return true
}

func showMessageRequestClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func showMessageRequestParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["message"]; !has {
		return false
	}
	if _, has := obj["type"]; !has {
		return false
	}
	return true
}

func signatureHelpMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["signatures"]; !has {
		return false
	}
	return true
}

func signatureHelpClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func signatureHelpContextMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["isRetrigger"]; !has {
		return false
	}
	if _, has := obj["triggerKind"]; !has {
		return false
	}
	return true
}

func signatureHelpOptionsMatches(raw any) bool {
	return isObject(raw)
}

func signatureHelpParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func signatureHelpRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func signatureInformationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["label"]; !has {
		return false
	}
	return true
}

func snippetTextEditMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["snippet"]; !has {
		return false
	}
	return true
}

func staleRequestSupportOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["cancel"]; !has {
		return false
	}
	if _, has := obj["retryOnContentModified"]; !has {
		return false
	}
	return true
}

func staticRegistrationOptionsMatches(raw any) bool {
	return isObject(raw)
}

func stringValueMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["value"]; !has {
		return false
	}
	return true
}

func symbolInformationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["location"]; !has {
		return false
	}
	if _, has := obj["name"]; !has {
		return false
	}
	return true
}

func textDocumentChangeRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	if _, has := obj["syncKind"]; !has {
		return false
	}
	return true
}

func textDocumentClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func textDocumentContentChangePartialMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["text"]; !has {
		return false
	}
	return true
}

func textDocumentContentChangeWholeDocumentMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["text"]; !has {
		return false
	}
	return true
}

func textDocumentContentClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func textDocumentContentOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["schemes"]; !has {
		return false
	}
	return true
}

func textDocumentContentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func textDocumentContentRefreshParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func textDocumentContentRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["schemes"]; !has {
		return false
	}
	return true
}

func textDocumentContentResultMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["text"]; !has {
		return false
	}
	return true
}

func textDocumentEditMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["edits"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func textDocumentFilterClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func textDocumentFilterLanguageMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["language"]; !has {
		return false
	}
	return true
}

func textDocumentFilterPatternMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["pattern"]; !has {
		return false
	}
	return true
}

func textDocumentFilterSchemeMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["scheme"]; !has {
		return false
	}
	return true
}

func textDocumentIdentifierMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func textDocumentItemMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["languageId"]; !has {
		return false
	}
	if _, has := obj["text"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	if _, has := obj["version"]; !has {
		return false
	}
	return true
}

func textDocumentPositionParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func textDocumentRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func textDocumentSaveRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func textDocumentSyncClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func textDocumentSyncOptionsMatches(raw any) bool {
	return isObject(raw)
}

func textEditMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["newText"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	return true
}

func typeDefinitionClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func typeDefinitionOptionsMatches(raw any) bool {
	return isObject(raw)
}

func typeDefinitionParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func typeDefinitionRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func typeHierarchyClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func typeHierarchyItemMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["name"]; !has {
		return false
	}
	if _, has := obj["range"]; !has {
		return false
	}
	if _, has := obj["selectionRange"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func typeHierarchyOptionsMatches(raw any) bool {
	return isObject(raw)
}

func typeHierarchyPrepareParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["position"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func typeHierarchyRegistrationOptionsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["documentSelector"]; !has {
		return false
	}
	return true
}

func typeHierarchySubtypesParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["item"]; !has {
		return false
	}
	return true
}

func typeHierarchySupertypesParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["item"]; !has {
		return false
	}
	return true
}

func unchangedDocumentDiagnosticReportMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["resultId"]; !has {
		return false
	}
	return true
}

func unregistrationMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["id"]; !has {
		return false
	}
	if _, has := obj["method"]; !has {
		return false
	}
	return true
}

func unregistrationParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["unregisterations"]; !has {
		return false
	}
	return true
}

func versionedNotebookDocumentIdentifierMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	if _, has := obj["version"]; !has {
		return false
	}
	return true
}

func versionedTextDocumentIdentifierMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	if _, has := obj["version"]; !has {
		return false
	}
	return true
}

func willSaveTextDocumentParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["reason"]; !has {
		return false
	}
	if _, has := obj["textDocument"]; !has {
		return false
	}
	return true
}

func windowClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func workDoneProgressBeginMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["title"]; !has {
		return false
	}
	return true
}

func workDoneProgressCancelParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["token"]; !has {
		return false
	}
	return true
}

func workDoneProgressCreateParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["token"]; !has {
		return false
	}
	return true
}

func workDoneProgressEndMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	return true
}

func workDoneProgressOptionsMatches(raw any) bool {
	return isObject(raw)
}

func workDoneProgressParamsMatches(raw any) bool {
	return isObject(raw)
}

func workDoneProgressReportMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	return true
}

func workspaceClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func workspaceDiagnosticParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["previousResultIds"]; !has {
		return false
	}
	return true
}

func workspaceDiagnosticReportMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["items"]; !has {
		return false
	}
	return true
}

func workspaceDiagnosticReportPartialResultMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["items"]; !has {
		return false
	}
	return true
}

func workspaceEditMatches(raw any) bool {
	return isObject(raw)
}

func workspaceEditClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func workspaceEditMetadataMatches(raw any) bool {
	return isObject(raw)
}

func workspaceFolderMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["name"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	return true
}

func workspaceFoldersChangeEventMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["added"]; !has {
		return false
	}
	if _, has := obj["removed"]; !has {
		return false
	}
	return true
}

func workspaceFoldersInitializeParamsMatches(raw any) bool {
	return isObject(raw)
}

func workspaceFoldersServerCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func workspaceFullDocumentDiagnosticReportMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["items"]; !has {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	if _, has := obj["version"]; !has {
		return false
	}
	return true
}

func workspaceOptionsMatches(raw any) bool {
	return isObject(raw)
}

func workspaceSymbolMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["location"]; !has {
		return false
	}
	if _, has := obj["name"]; !has {
		return false
	}
	return true
}

func workspaceSymbolClientCapabilitiesMatches(raw any) bool {
	return isObject(raw)
}

func workspaceSymbolOptionsMatches(raw any) bool {
	return isObject(raw)
}

func workspaceSymbolParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["query"]; !has {
		return false
	}
	return true
}

func workspaceSymbolRegistrationOptionsMatches(raw any) bool {
	return isObject(raw)
}

func workspaceUnchangedDocumentDiagnosticReportMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["kind"]; !has {
		return false
	}
	if _, has := obj["resultId"]; !has {
		return false
	}
	if _, has := obj["uri"]; !has {
		return false
	}
	if _, has := obj["version"]; !has {
		return false
	}
	return true
}

func _InitializeParamsMatches(raw any) bool {
	obj, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	if _, has := obj["capabilities"]; !has {
		return false
	}
	if _, has := obj["processId"]; !has {
		return false
	}
	if _, has := obj["rootUri"]; !has {
		return false
	}
	return true
}

func applyKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	return false
}

func codeActionKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "" {
		return true
	}
	if s, ok := raw.(string); ok && s == "quickfix" {
		return true
	}
	if s, ok := raw.(string); ok && s == "refactor" {
		return true
	}
	if s, ok := raw.(string); ok && s == "refactor.extract" {
		return true
	}
	if s, ok := raw.(string); ok && s == "refactor.inline" {
		return true
	}
	if s, ok := raw.(string); ok && s == "refactor.move" {
		return true
	}
	if s, ok := raw.(string); ok && s == "refactor.rewrite" {
		return true
	}
	if s, ok := raw.(string); ok && s == "source" {
		return true
	}
	if s, ok := raw.(string); ok && s == "source.organizeImports" {
		return true
	}
	if s, ok := raw.(string); ok && s == "source.fixAll" {
		return true
	}
	if s, ok := raw.(string); ok && s == "notebook" {
		return true
	}
	return false
}

func codeActionTagMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	return false
}

func codeActionTriggerKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	return false
}

func completionItemKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 3 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 4 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 5 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 6 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 7 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 8 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 9 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 10 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 11 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 12 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 13 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 14 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 15 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 16 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 17 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 18 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 19 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 20 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 21 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 22 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 23 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 24 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 25 {
		return true
	}
	return false
}

func completionItemTagMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	return false
}

func completionTriggerKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 3 {
		return true
	}
	return false
}

func diagnosticSeverityMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 3 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 4 {
		return true
	}
	return false
}

func diagnosticTagMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	return false
}

func documentDiagnosticReportKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "full" {
		return true
	}
	if s, ok := raw.(string); ok && s == "unchanged" {
		return true
	}
	return false
}

func documentHighlightKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 3 {
		return true
	}
	return false
}

func errorCodesMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == -32700 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == -32600 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == -32601 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == -32602 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == -32603 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == -32002 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == -32001 {
		return true
	}
	return false
}

func failureHandlingKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "abort" {
		return true
	}
	if s, ok := raw.(string); ok && s == "transactional" {
		return true
	}
	if s, ok := raw.(string); ok && s == "textOnlyTransactional" {
		return true
	}
	if s, ok := raw.(string); ok && s == "undo" {
		return true
	}
	return false
}

func fileChangeTypeMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 3 {
		return true
	}
	return false
}

func fileOperationPatternKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "file" {
		return true
	}
	if s, ok := raw.(string); ok && s == "folder" {
		return true
	}
	return false
}

func foldingRangeKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "comment" {
		return true
	}
	if s, ok := raw.(string); ok && s == "imports" {
		return true
	}
	if s, ok := raw.(string); ok && s == "region" {
		return true
	}
	return false
}

func inlayHintKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	return false
}

func inlineCompletionTriggerKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	return false
}

func insertTextFormatMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	return false
}

func insertTextModeMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	return false
}

func lSPErrorCodesMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == -32803 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == -32802 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == -32801 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == -32800 {
		return true
	}
	return false
}

func languageKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "abap" {
		return true
	}
	if s, ok := raw.(string); ok && s == "bat" {
		return true
	}
	if s, ok := raw.(string); ok && s == "bibtex" {
		return true
	}
	if s, ok := raw.(string); ok && s == "clojure" {
		return true
	}
	if s, ok := raw.(string); ok && s == "coffeescript" {
		return true
	}
	if s, ok := raw.(string); ok && s == "c" {
		return true
	}
	if s, ok := raw.(string); ok && s == "cpp" {
		return true
	}
	if s, ok := raw.(string); ok && s == "csharp" {
		return true
	}
	if s, ok := raw.(string); ok && s == "css" {
		return true
	}
	if s, ok := raw.(string); ok && s == "d" {
		return true
	}
	if s, ok := raw.(string); ok && s == "pascal" {
		return true
	}
	if s, ok := raw.(string); ok && s == "diff" {
		return true
	}
	if s, ok := raw.(string); ok && s == "dart" {
		return true
	}
	if s, ok := raw.(string); ok && s == "dockerfile" {
		return true
	}
	if s, ok := raw.(string); ok && s == "elixir" {
		return true
	}
	if s, ok := raw.(string); ok && s == "erlang" {
		return true
	}
	if s, ok := raw.(string); ok && s == "fsharp" {
		return true
	}
	if s, ok := raw.(string); ok && s == "git-commit" {
		return true
	}
	if s, ok := raw.(string); ok && s == "git-rebase" {
		return true
	}
	if s, ok := raw.(string); ok && s == "go" {
		return true
	}
	if s, ok := raw.(string); ok && s == "groovy" {
		return true
	}
	if s, ok := raw.(string); ok && s == "handlebars" {
		return true
	}
	if s, ok := raw.(string); ok && s == "haskell" {
		return true
	}
	if s, ok := raw.(string); ok && s == "html" {
		return true
	}
	if s, ok := raw.(string); ok && s == "ini" {
		return true
	}
	if s, ok := raw.(string); ok && s == "java" {
		return true
	}
	if s, ok := raw.(string); ok && s == "javascript" {
		return true
	}
	if s, ok := raw.(string); ok && s == "javascriptreact" {
		return true
	}
	if s, ok := raw.(string); ok && s == "json" {
		return true
	}
	if s, ok := raw.(string); ok && s == "latex" {
		return true
	}
	if s, ok := raw.(string); ok && s == "less" {
		return true
	}
	if s, ok := raw.(string); ok && s == "lua" {
		return true
	}
	if s, ok := raw.(string); ok && s == "makefile" {
		return true
	}
	if s, ok := raw.(string); ok && s == "markdown" {
		return true
	}
	if s, ok := raw.(string); ok && s == "objective-c" {
		return true
	}
	if s, ok := raw.(string); ok && s == "objective-cpp" {
		return true
	}
	if s, ok := raw.(string); ok && s == "pascal" {
		return true
	}
	if s, ok := raw.(string); ok && s == "perl" {
		return true
	}
	if s, ok := raw.(string); ok && s == "perl6" {
		return true
	}
	if s, ok := raw.(string); ok && s == "php" {
		return true
	}
	if s, ok := raw.(string); ok && s == "plaintext" {
		return true
	}
	if s, ok := raw.(string); ok && s == "powershell" {
		return true
	}
	if s, ok := raw.(string); ok && s == "jade" {
		return true
	}
	if s, ok := raw.(string); ok && s == "python" {
		return true
	}
	if s, ok := raw.(string); ok && s == "r" {
		return true
	}
	if s, ok := raw.(string); ok && s == "razor" {
		return true
	}
	if s, ok := raw.(string); ok && s == "ruby" {
		return true
	}
	if s, ok := raw.(string); ok && s == "rust" {
		return true
	}
	if s, ok := raw.(string); ok && s == "scss" {
		return true
	}
	if s, ok := raw.(string); ok && s == "sass" {
		return true
	}
	if s, ok := raw.(string); ok && s == "scala" {
		return true
	}
	if s, ok := raw.(string); ok && s == "shaderlab" {
		return true
	}
	if s, ok := raw.(string); ok && s == "shellscript" {
		return true
	}
	if s, ok := raw.(string); ok && s == "sql" {
		return true
	}
	if s, ok := raw.(string); ok && s == "swift" {
		return true
	}
	if s, ok := raw.(string); ok && s == "typescript" {
		return true
	}
	if s, ok := raw.(string); ok && s == "typescriptreact" {
		return true
	}
	if s, ok := raw.(string); ok && s == "tex" {
		return true
	}
	if s, ok := raw.(string); ok && s == "vb" {
		return true
	}
	if s, ok := raw.(string); ok && s == "xml" {
		return true
	}
	if s, ok := raw.(string); ok && s == "xsl" {
		return true
	}
	if s, ok := raw.(string); ok && s == "yaml" {
		return true
	}
	return false
}

func markupKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "plaintext" {
		return true
	}
	if s, ok := raw.(string); ok && s == "markdown" {
		return true
	}
	return false
}

func messageTypeMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 3 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 4 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 5 {
		return true
	}
	return false
}

func monikerKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "import" {
		return true
	}
	if s, ok := raw.(string); ok && s == "export" {
		return true
	}
	if s, ok := raw.(string); ok && s == "local" {
		return true
	}
	return false
}

func notebookCellKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	return false
}

func positionEncodingKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "utf-8" {
		return true
	}
	if s, ok := raw.(string); ok && s == "utf-16" {
		return true
	}
	if s, ok := raw.(string); ok && s == "utf-32" {
		return true
	}
	return false
}

func prepareSupportDefaultBehaviorMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	return false
}

func resourceOperationKindMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "create" {
		return true
	}
	if s, ok := raw.(string); ok && s == "rename" {
		return true
	}
	if s, ok := raw.(string); ok && s == "delete" {
		return true
	}
	return false
}

func semanticTokenModifiersMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "declaration" {
		return true
	}
	if s, ok := raw.(string); ok && s == "definition" {
		return true
	}
	if s, ok := raw.(string); ok && s == "readonly" {
		return true
	}
	if s, ok := raw.(string); ok && s == "static" {
		return true
	}
	if s, ok := raw.(string); ok && s == "deprecated" {
		return true
	}
	if s, ok := raw.(string); ok && s == "abstract" {
		return true
	}
	if s, ok := raw.(string); ok && s == "async" {
		return true
	}
	if s, ok := raw.(string); ok && s == "modification" {
		return true
	}
	if s, ok := raw.(string); ok && s == "documentation" {
		return true
	}
	if s, ok := raw.(string); ok && s == "defaultLibrary" {
		return true
	}
	return false
}

func semanticTokenTypesMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "namespace" {
		return true
	}
	if s, ok := raw.(string); ok && s == "type" {
		return true
	}
	if s, ok := raw.(string); ok && s == "class" {
		return true
	}
	if s, ok := raw.(string); ok && s == "enum" {
		return true
	}
	if s, ok := raw.(string); ok && s == "interface" {
		return true
	}
	if s, ok := raw.(string); ok && s == "struct" {
		return true
	}
	if s, ok := raw.(string); ok && s == "typeParameter" {
		return true
	}
	if s, ok := raw.(string); ok && s == "parameter" {
		return true
	}
	if s, ok := raw.(string); ok && s == "variable" {
		return true
	}
	if s, ok := raw.(string); ok && s == "property" {
		return true
	}
	if s, ok := raw.(string); ok && s == "enumMember" {
		return true
	}
	if s, ok := raw.(string); ok && s == "event" {
		return true
	}
	if s, ok := raw.(string); ok && s == "function" {
		return true
	}
	if s, ok := raw.(string); ok && s == "method" {
		return true
	}
	if s, ok := raw.(string); ok && s == "macro" {
		return true
	}
	if s, ok := raw.(string); ok && s == "keyword" {
		return true
	}
	if s, ok := raw.(string); ok && s == "modifier" {
		return true
	}
	if s, ok := raw.(string); ok && s == "comment" {
		return true
	}
	if s, ok := raw.(string); ok && s == "string" {
		return true
	}
	if s, ok := raw.(string); ok && s == "number" {
		return true
	}
	if s, ok := raw.(string); ok && s == "regexp" {
		return true
	}
	if s, ok := raw.(string); ok && s == "operator" {
		return true
	}
	if s, ok := raw.(string); ok && s == "decorator" {
		return true
	}
	if s, ok := raw.(string); ok && s == "label" {
		return true
	}
	return false
}

func signatureHelpTriggerKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 3 {
		return true
	}
	return false
}

func symbolKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 3 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 4 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 5 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 6 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 7 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 8 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 9 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 10 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 11 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 12 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 13 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 14 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 15 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 16 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 17 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 18 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 19 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 20 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 21 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 22 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 23 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 24 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 25 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 26 {
		return true
	}
	return false
}

func symbolTagMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	return false
}

func textDocumentSaveReasonMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 3 {
		return true
	}
	return false
}

func textDocumentSyncKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 0 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	return false
}

func tokenFormatMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "relative" {
		return true
	}
	return false
}

func traceValueMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "off" {
		return true
	}
	if s, ok := raw.(string); ok && s == "messages" {
		return true
	}
	if s, ok := raw.(string); ok && s == "verbose" {
		return true
	}
	return false
}

func uniquenessLevelMatches(raw any) bool {
	if s, ok := raw.(string); ok && s == "document" {
		return true
	}
	if s, ok := raw.(string); ok && s == "project" {
		return true
	}
	if s, ok := raw.(string); ok && s == "group" {
		return true
	}
	if s, ok := raw.(string); ok && s == "scheme" {
		return true
	}
	if s, ok := raw.(string); ok && s == "global" {
		return true
	}
	return false
}

func watchKindMatches(raw any) bool {
	if n, ok := raw.(float64); ok && int64(n) == 1 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 2 {
		return true
	}
	if n, ok := raw.(float64); ok && int64(n) == 4 {
		return true
	}
	return false
}

func changeAnnotationIdentifierMatches(raw any) bool {
	return isString(raw)
}

func declarationMatches(raw any) bool {
	if locationMatches(raw) {
		return true
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || locationMatches(raw.([]any)[0])) {
		return true
	}
	return false
}

func declarationLinkMatches(raw any) bool {
	return locationLinkMatches(raw)
}

func definitionMatches(raw any) bool {
	if locationMatches(raw) {
		return true
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || locationMatches(raw.([]any)[0])) {
		return true
	}
	return false
}

func definitionLinkMatches(raw any) bool {
	return locationLinkMatches(raw)
}

func documentDiagnosticReportMatches(raw any) bool {
	if relatedFullDocumentDiagnosticReportMatches(raw) {
		return true
	}
	if relatedUnchangedDocumentDiagnosticReportMatches(raw) {
		return true
	}
	return false
}

func documentDiagnosticReportProgressMatches(raw any) bool {
	if documentDiagnosticReportMatches(raw) {
		return true
	}
	if documentDiagnosticReportPartialResultMatches(raw) {
		return true
	}
	return false
}

func documentFilterMatches(raw any) bool {
	if textDocumentFilterMatches(raw) {
		return true
	}
	if notebookCellTextDocumentFilterMatches(raw) {
		return true
	}
	return false
}

func documentSelectorMatches(raw any) bool {
	return isArray(raw) && (len(raw.([]any)) == 0 || documentFilterMatches(raw.([]any)[0]))
}

func globPatternMatches(raw any) bool {
	if patternMatches(raw) {
		return true
	}
	if relativePatternMatches(raw) {
		return true
	}
	return false
}

func inlineValueMatches(raw any) bool {
	if inlineValueTextMatches(raw) {
		return true
	}
	if inlineValueVariableLookupMatches(raw) {
		return true
	}
	if inlineValueEvaluatableExpressionMatches(raw) {
		return true
	}
	return false
}

func lSPAnyMatches(raw any) bool {
	if lSPObjectMatches(raw) {
		return true
	}
	if lSPArrayMatches(raw) {
		return true
	}
	if isString(raw) {
		return true
	}
	if isNumber(raw) {
		return true
	}
	if isNumber(raw) {
		return true
	}
	if isNumber(raw) {
		return true
	}
	if isBool(raw) {
		return true
	}
	return false
}

func lSPArrayMatches(raw any) bool {
	return isArray(raw) && (len(raw.([]any)) == 0 || lSPAnyMatches(raw.([]any)[0]))
}

func lSPObjectMatches(raw any) bool {
	return isObject(raw)
}

func markedStringMatches(raw any) bool {
	if isString(raw) {
		return true
	}
	if markedStringWithLanguageMatches(raw) {
		return true
	}
	return false
}

func notebookDocumentFilterMatches(raw any) bool {
	if notebookDocumentFilterNotebookTypeMatches(raw) {
		return true
	}
	if notebookDocumentFilterSchemeMatches(raw) {
		return true
	}
	if notebookDocumentFilterPatternMatches(raw) {
		return true
	}
	return false
}

func patternMatches(raw any) bool {
	return isString(raw)
}

func prepareRenameResultMatches(raw any) bool {
	if rangeMatches(raw) {
		return true
	}
	if prepareRenamePlaceholderMatches(raw) {
		return true
	}
	if prepareRenameDefaultBehaviorMatches(raw) {
		return true
	}
	return false
}

func progressTokenMatches(raw any) bool {
	if isNumber(raw) {
		return true
	}
	if isString(raw) {
		return true
	}
	return false
}

func regularExpressionEngineKindMatches(raw any) bool {
	return isString(raw)
}

func textDocumentContentChangeEventMatches(raw any) bool {
	if textDocumentContentChangePartialMatches(raw) {
		return true
	}
	if textDocumentContentChangeWholeDocumentMatches(raw) {
		return true
	}
	return false
}

func textDocumentFilterMatches(raw any) bool {
	if textDocumentFilterLanguageMatches(raw) {
		return true
	}
	if textDocumentFilterSchemeMatches(raw) {
		return true
	}
	if textDocumentFilterPatternMatches(raw) {
		return true
	}
	return false
}

func workspaceDocumentDiagnosticReportMatches(raw any) bool {
	if workspaceFullDocumentDiagnosticReportMatches(raw) {
		return true
	}
	if workspaceUnchangedDocumentDiagnosticReportMatches(raw) {
		return true
	}
	return false
}

func (u Definition) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *Definition) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if locationMatches(raw) {
		var v Location
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDefinitionLocation(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || locationMatches(raw.([]any)[0])) {
		var v []Location
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDefinitionArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Location returns the Location variant value and true if selected.
func (u Definition) Location() (Location, bool) {
	if u.tag != 0 {
		var zero Location
		return zero, false
	}
	return u.value.(Location), true
}

// Array1 returns the Location[] variant value and true if selected.
func (u Definition) Array1() ([]Location, bool) {
	if u.tag != 1 {
		var zero []Location
		return zero, false
	}
	return u.value.([]Location), true
}

func (u LSPAny) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	case 3:
		return json.Marshal(u.value)
	case 4:
		return json.Marshal(u.value)
	case 5:
		return json.Marshal(u.value)
	case 6:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *LSPAny) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if lSPObjectMatches(raw) {
		var v LSPObject
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewLSPAnyLSPObject(v)
		return nil
	}
	if lSPArrayMatches(raw) {
		var v LSPArray
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewLSPAnyLSPArray(v)
		return nil
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewLSPAnyString(v)
		return nil
	}
	if isNumber(raw) {
		var v int32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewLSPAnyInteger(v)
		return nil
	}
	if isNumber(raw) {
		var v uint32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewLSPAnyUinteger(v)
		return nil
	}
	if isNumber(raw) {
		var v float64
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewLSPAnyDecimal(v)
		return nil
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewLSPAnyBoolean(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// LSPObject returns the LSPObject variant value and true if selected.
func (u LSPAny) LSPObject() (LSPObject, bool) {
	if u.tag != 0 {
		var zero LSPObject
		return zero, false
	}
	return u.value.(LSPObject), true
}

// LSPArray returns the LSPArray variant value and true if selected.
func (u LSPAny) LSPArray() (LSPArray, bool) {
	if u.tag != 1 {
		var zero LSPArray
		return zero, false
	}
	return u.value.(LSPArray), true
}

// String returns the string variant value and true if selected.
func (u LSPAny) String() (string, bool) {
	if u.tag != 2 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// Integer returns the integer variant value and true if selected.
func (u LSPAny) Integer() (int32, bool) {
	if u.tag != 3 {
		var zero int32
		return zero, false
	}
	return u.value.(int32), true
}

// Uinteger returns the uinteger variant value and true if selected.
func (u LSPAny) Uinteger() (uint32, bool) {
	if u.tag != 4 {
		var zero uint32
		return zero, false
	}
	return u.value.(uint32), true
}

// Decimal returns the decimal variant value and true if selected.
func (u LSPAny) Decimal() (float64, bool) {
	if u.tag != 5 {
		var zero float64
		return zero, false
	}
	return u.value.(float64), true
}

// Boolean returns the boolean variant value and true if selected.
func (u LSPAny) Boolean() (bool, bool) {
	if u.tag != 6 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

func (u Declaration) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *Declaration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if locationMatches(raw) {
		var v Location
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDeclarationLocation(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || locationMatches(raw.([]any)[0])) {
		var v []Location
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDeclarationArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Location returns the Location variant value and true if selected.
func (u Declaration) Location() (Location, bool) {
	if u.tag != 0 {
		var zero Location
		return zero, false
	}
	return u.value.(Location), true
}

// Array1 returns the Location[] variant value and true if selected.
func (u Declaration) Array1() ([]Location, bool) {
	if u.tag != 1 {
		var zero []Location
		return zero, false
	}
	return u.value.([]Location), true
}

func (u InlineValue) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *InlineValue) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if inlineValueTextMatches(raw) {
		var v InlineValueText
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewInlineValueInlineValueText(v)
		return nil
	}
	if inlineValueVariableLookupMatches(raw) {
		var v InlineValueVariableLookup
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewInlineValueInlineValueVariableLookup(v)
		return nil
	}
	if inlineValueEvaluatableExpressionMatches(raw) {
		var v InlineValueEvaluatableExpression
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewInlineValueInlineValueEvaluatableExpression(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// InlineValueText returns the InlineValueText variant value and true if selected.
func (u InlineValue) InlineValueText() (InlineValueText, bool) {
	if u.tag != 0 {
		var zero InlineValueText
		return zero, false
	}
	return u.value.(InlineValueText), true
}

// InlineValueVariableLookup returns the InlineValueVariableLookup variant value and true if selected.
func (u InlineValue) InlineValueVariableLookup() (InlineValueVariableLookup, bool) {
	if u.tag != 1 {
		var zero InlineValueVariableLookup
		return zero, false
	}
	return u.value.(InlineValueVariableLookup), true
}

// InlineValueEvaluatableExpression returns the InlineValueEvaluatableExpression variant value and true if selected.
func (u InlineValue) InlineValueEvaluatableExpression() (InlineValueEvaluatableExpression, bool) {
	if u.tag != 2 {
		var zero InlineValueEvaluatableExpression
		return zero, false
	}
	return u.value.(InlineValueEvaluatableExpression), true
}

func (u DocumentDiagnosticReport) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *DocumentDiagnosticReport) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if relatedFullDocumentDiagnosticReportMatches(raw) {
		var v RelatedFullDocumentDiagnosticReport
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDocumentDiagnosticReportRelatedFullDocumentDiagnosticReport(v)
		return nil
	}
	if relatedUnchangedDocumentDiagnosticReportMatches(raw) {
		var v RelatedUnchangedDocumentDiagnosticReport
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDocumentDiagnosticReportRelatedUnchangedDocumentDiagnosticReport(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// RelatedFullDocumentDiagnosticReport returns the RelatedFullDocumentDiagnosticReport variant value and true if selected.
func (u DocumentDiagnosticReport) RelatedFullDocumentDiagnosticReport() (RelatedFullDocumentDiagnosticReport, bool) {
	if u.tag != 0 {
		var zero RelatedFullDocumentDiagnosticReport
		return zero, false
	}
	return u.value.(RelatedFullDocumentDiagnosticReport), true
}

// RelatedUnchangedDocumentDiagnosticReport returns the RelatedUnchangedDocumentDiagnosticReport variant value and true if selected.
func (u DocumentDiagnosticReport) RelatedUnchangedDocumentDiagnosticReport() (RelatedUnchangedDocumentDiagnosticReport, bool) {
	if u.tag != 1 {
		var zero RelatedUnchangedDocumentDiagnosticReport
		return zero, false
	}
	return u.value.(RelatedUnchangedDocumentDiagnosticReport), true
}

func (u DocumentDiagnosticReportProgress) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *DocumentDiagnosticReportProgress) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if documentDiagnosticReportMatches(raw) {
		var v DocumentDiagnosticReport
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDocumentDiagnosticReportProgressDocumentDiagnosticReport(v)
		return nil
	}
	if documentDiagnosticReportPartialResultMatches(raw) {
		var v DocumentDiagnosticReportPartialResult
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDocumentDiagnosticReportProgressDocumentDiagnosticReportPartialResult(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// DocumentDiagnosticReport returns the DocumentDiagnosticReport variant value and true if selected.
func (u DocumentDiagnosticReportProgress) DocumentDiagnosticReport() (DocumentDiagnosticReport, bool) {
	if u.tag != 0 {
		var zero DocumentDiagnosticReport
		return zero, false
	}
	return u.value.(DocumentDiagnosticReport), true
}

// DocumentDiagnosticReportPartialResult returns the DocumentDiagnosticReportPartialResult variant value and true if selected.
func (u DocumentDiagnosticReportProgress) DocumentDiagnosticReportPartialResult() (DocumentDiagnosticReportPartialResult, bool) {
	if u.tag != 1 {
		var zero DocumentDiagnosticReportPartialResult
		return zero, false
	}
	return u.value.(DocumentDiagnosticReportPartialResult), true
}

func (u PrepareRenameResult) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *PrepareRenameResult) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if rangeMatches(raw) {
		var v Range
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewPrepareRenameResultRange(v)
		return nil
	}
	if prepareRenamePlaceholderMatches(raw) {
		var v PrepareRenamePlaceholder
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewPrepareRenameResultPrepareRenamePlaceholder(v)
		return nil
	}
	if prepareRenameDefaultBehaviorMatches(raw) {
		var v PrepareRenameDefaultBehavior
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewPrepareRenameResultPrepareRenameDefaultBehavior(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Range returns the Range variant value and true if selected.
func (u PrepareRenameResult) Range() (Range, bool) {
	if u.tag != 0 {
		var zero Range
		return zero, false
	}
	return u.value.(Range), true
}

// PrepareRenamePlaceholder returns the PrepareRenamePlaceholder variant value and true if selected.
func (u PrepareRenameResult) PrepareRenamePlaceholder() (PrepareRenamePlaceholder, bool) {
	if u.tag != 1 {
		var zero PrepareRenamePlaceholder
		return zero, false
	}
	return u.value.(PrepareRenamePlaceholder), true
}

// PrepareRenameDefaultBehavior returns the PrepareRenameDefaultBehavior variant value and true if selected.
func (u PrepareRenameResult) PrepareRenameDefaultBehavior() (PrepareRenameDefaultBehavior, bool) {
	if u.tag != 2 {
		var zero PrepareRenameDefaultBehavior
		return zero, false
	}
	return u.value.(PrepareRenameDefaultBehavior), true
}

func (u ProgressToken) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *ProgressToken) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isNumber(raw) {
		var v int32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewProgressTokenInteger(v)
		return nil
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewProgressTokenString(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Integer returns the integer variant value and true if selected.
func (u ProgressToken) Integer() (int32, bool) {
	if u.tag != 0 {
		var zero int32
		return zero, false
	}
	return u.value.(int32), true
}

// String returns the string variant value and true if selected.
func (u ProgressToken) String() (string, bool) {
	if u.tag != 1 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

func (u WorkspaceDocumentDiagnosticReport) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *WorkspaceDocumentDiagnosticReport) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if workspaceFullDocumentDiagnosticReportMatches(raw) {
		var v WorkspaceFullDocumentDiagnosticReport
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewWorkspaceDocumentDiagnosticReportWorkspaceFullDocumentDiagnosticReport(v)
		return nil
	}
	if workspaceUnchangedDocumentDiagnosticReportMatches(raw) {
		var v WorkspaceUnchangedDocumentDiagnosticReport
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewWorkspaceDocumentDiagnosticReportWorkspaceUnchangedDocumentDiagnosticReport(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// WorkspaceFullDocumentDiagnosticReport returns the WorkspaceFullDocumentDiagnosticReport variant value and true if selected.
func (u WorkspaceDocumentDiagnosticReport) WorkspaceFullDocumentDiagnosticReport() (WorkspaceFullDocumentDiagnosticReport, bool) {
	if u.tag != 0 {
		var zero WorkspaceFullDocumentDiagnosticReport
		return zero, false
	}
	return u.value.(WorkspaceFullDocumentDiagnosticReport), true
}

// WorkspaceUnchangedDocumentDiagnosticReport returns the WorkspaceUnchangedDocumentDiagnosticReport variant value and true if selected.
func (u WorkspaceDocumentDiagnosticReport) WorkspaceUnchangedDocumentDiagnosticReport() (WorkspaceUnchangedDocumentDiagnosticReport, bool) {
	if u.tag != 1 {
		var zero WorkspaceUnchangedDocumentDiagnosticReport
		return zero, false
	}
	return u.value.(WorkspaceUnchangedDocumentDiagnosticReport), true
}

func (u TextDocumentContentChangeEvent) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *TextDocumentContentChangeEvent) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if textDocumentContentChangePartialMatches(raw) {
		var v TextDocumentContentChangePartial
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewTextDocumentContentChangeEventTextDocumentContentChangePartial(v)
		return nil
	}
	if textDocumentContentChangeWholeDocumentMatches(raw) {
		var v TextDocumentContentChangeWholeDocument
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewTextDocumentContentChangeEventTextDocumentContentChangeWholeDocument(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// TextDocumentContentChangePartial returns the TextDocumentContentChangePartial variant value and true if selected.
func (u TextDocumentContentChangeEvent) TextDocumentContentChangePartial() (TextDocumentContentChangePartial, bool) {
	if u.tag != 0 {
		var zero TextDocumentContentChangePartial
		return zero, false
	}
	return u.value.(TextDocumentContentChangePartial), true
}

// TextDocumentContentChangeWholeDocument returns the TextDocumentContentChangeWholeDocument variant value and true if selected.
func (u TextDocumentContentChangeEvent) TextDocumentContentChangeWholeDocument() (TextDocumentContentChangeWholeDocument, bool) {
	if u.tag != 1 {
		var zero TextDocumentContentChangeWholeDocument
		return zero, false
	}
	return u.value.(TextDocumentContentChangeWholeDocument), true
}

func (u MarkedString) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *MarkedString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewMarkedStringString(v)
		return nil
	}
	if markedStringWithLanguageMatches(raw) {
		var v MarkedStringWithLanguage
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewMarkedStringMarkedStringWithLanguage(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u MarkedString) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// MarkedStringWithLanguage returns the MarkedStringWithLanguage variant value and true if selected.
func (u MarkedString) MarkedStringWithLanguage() (MarkedStringWithLanguage, bool) {
	if u.tag != 1 {
		var zero MarkedStringWithLanguage
		return zero, false
	}
	return u.value.(MarkedStringWithLanguage), true
}

func (u DocumentFilter) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *DocumentFilter) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if textDocumentFilterMatches(raw) {
		var v TextDocumentFilter
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDocumentFilterTextDocumentFilter(v)
		return nil
	}
	if notebookCellTextDocumentFilterMatches(raw) {
		var v NotebookCellTextDocumentFilter
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewDocumentFilterNotebookCellTextDocumentFilter(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// TextDocumentFilter returns the TextDocumentFilter variant value and true if selected.
func (u DocumentFilter) TextDocumentFilter() (TextDocumentFilter, bool) {
	if u.tag != 0 {
		var zero TextDocumentFilter
		return zero, false
	}
	return u.value.(TextDocumentFilter), true
}

// NotebookCellTextDocumentFilter returns the NotebookCellTextDocumentFilter variant value and true if selected.
func (u DocumentFilter) NotebookCellTextDocumentFilter() (NotebookCellTextDocumentFilter, bool) {
	if u.tag != 1 {
		var zero NotebookCellTextDocumentFilter
		return zero, false
	}
	return u.value.(NotebookCellTextDocumentFilter), true
}

func (u GlobPattern) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *GlobPattern) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if patternMatches(raw) {
		var v Pattern
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewGlobPatternPattern(v)
		return nil
	}
	if relativePatternMatches(raw) {
		var v RelativePattern
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewGlobPatternRelativePattern(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Pattern returns the Pattern variant value and true if selected.
func (u GlobPattern) Pattern() (Pattern, bool) {
	if u.tag != 0 {
		var zero Pattern
		return zero, false
	}
	return u.value.(Pattern), true
}

// RelativePattern returns the RelativePattern variant value and true if selected.
func (u GlobPattern) RelativePattern() (RelativePattern, bool) {
	if u.tag != 1 {
		var zero RelativePattern
		return zero, false
	}
	return u.value.(RelativePattern), true
}

func (u TextDocumentFilter) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *TextDocumentFilter) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if textDocumentFilterLanguageMatches(raw) {
		var v TextDocumentFilterLanguage
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewTextDocumentFilterTextDocumentFilterLanguage(v)
		return nil
	}
	if textDocumentFilterSchemeMatches(raw) {
		var v TextDocumentFilterScheme
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewTextDocumentFilterTextDocumentFilterScheme(v)
		return nil
	}
	if textDocumentFilterPatternMatches(raw) {
		var v TextDocumentFilterPattern
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewTextDocumentFilterTextDocumentFilterPattern(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// TextDocumentFilterLanguage returns the TextDocumentFilterLanguage variant value and true if selected.
func (u TextDocumentFilter) TextDocumentFilterLanguage() (TextDocumentFilterLanguage, bool) {
	if u.tag != 0 {
		var zero TextDocumentFilterLanguage
		return zero, false
	}
	return u.value.(TextDocumentFilterLanguage), true
}

// TextDocumentFilterScheme returns the TextDocumentFilterScheme variant value and true if selected.
func (u TextDocumentFilter) TextDocumentFilterScheme() (TextDocumentFilterScheme, bool) {
	if u.tag != 1 {
		var zero TextDocumentFilterScheme
		return zero, false
	}
	return u.value.(TextDocumentFilterScheme), true
}

// TextDocumentFilterPattern returns the TextDocumentFilterPattern variant value and true if selected.
func (u TextDocumentFilter) TextDocumentFilterPattern() (TextDocumentFilterPattern, bool) {
	if u.tag != 2 {
		var zero TextDocumentFilterPattern
		return zero, false
	}
	return u.value.(TextDocumentFilterPattern), true
}

func (u NotebookDocumentFilter) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *NotebookDocumentFilter) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if notebookDocumentFilterNotebookTypeMatches(raw) {
		var v NotebookDocumentFilterNotebookType
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewNotebookDocumentFilterNotebookDocumentFilterNotebookType(v)
		return nil
	}
	if notebookDocumentFilterSchemeMatches(raw) {
		var v NotebookDocumentFilterScheme
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewNotebookDocumentFilterNotebookDocumentFilterScheme(v)
		return nil
	}
	if notebookDocumentFilterPatternMatches(raw) {
		var v NotebookDocumentFilterPattern
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewNotebookDocumentFilterNotebookDocumentFilterPattern(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// NotebookDocumentFilterNotebookType returns the NotebookDocumentFilterNotebookType variant value and true if selected.
func (u NotebookDocumentFilter) NotebookDocumentFilterNotebookType() (NotebookDocumentFilterNotebookType, bool) {
	if u.tag != 0 {
		var zero NotebookDocumentFilterNotebookType
		return zero, false
	}
	return u.value.(NotebookDocumentFilterNotebookType), true
}

// NotebookDocumentFilterScheme returns the NotebookDocumentFilterScheme variant value and true if selected.
func (u NotebookDocumentFilter) NotebookDocumentFilterScheme() (NotebookDocumentFilterScheme, bool) {
	if u.tag != 1 {
		var zero NotebookDocumentFilterScheme
		return zero, false
	}
	return u.value.(NotebookDocumentFilterScheme), true
}

// NotebookDocumentFilterPattern returns the NotebookDocumentFilterPattern variant value and true if selected.
func (u NotebookDocumentFilter) NotebookDocumentFilterPattern() (NotebookDocumentFilterPattern, bool) {
	if u.tag != 2 {
		var zero NotebookDocumentFilterPattern
		return zero, false
	}
	return u.value.(NotebookDocumentFilterPattern), true
}

func (u OrCancelParamsId) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrCancelParamsId) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isNumber(raw) {
		var v int32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrCancelParamsIdInteger(v)
		return nil
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrCancelParamsIdString(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Integer returns the integer variant value and true if selected.
func (u OrCancelParamsId) Integer() (int32, bool) {
	if u.tag != 0 {
		var zero int32
		return zero, false
	}
	return u.value.(int32), true
}

// String returns the string variant value and true if selected.
func (u OrCancelParamsId) String() (string, bool) {
	if u.tag != 1 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

func (u OrClientSemanticTokensRequestOptionsRange) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrClientSemanticTokensRequestOptionsRange) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrClientSemanticTokensRequestOptionsRangeBoolean(v)
		return nil
	}
	if isObject(raw) {
		var v LitClientSemanticTokensRequestOptionsRangeItem1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrClientSemanticTokensRequestOptionsRangeLiteral1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrClientSemanticTokensRequestOptionsRange) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// Literal1 returns the literal variant value and true if selected.
func (u OrClientSemanticTokensRequestOptionsRange) Literal1() (LitClientSemanticTokensRequestOptionsRangeItem1, bool) {
	if u.tag != 1 {
		var zero LitClientSemanticTokensRequestOptionsRangeItem1
		return zero, false
	}
	return u.value.(LitClientSemanticTokensRequestOptionsRangeItem1), true
}

func (u OrClientSemanticTokensRequestOptionsFull) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrClientSemanticTokensRequestOptionsFull) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrClientSemanticTokensRequestOptionsFullBoolean(v)
		return nil
	}
	if clientSemanticTokensRequestFullDeltaMatches(raw) {
		var v ClientSemanticTokensRequestFullDelta
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrClientSemanticTokensRequestOptionsFullClientSemanticTokensRequestFullDelta(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrClientSemanticTokensRequestOptionsFull) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// ClientSemanticTokensRequestFullDelta returns the ClientSemanticTokensRequestFullDelta variant value and true if selected.
func (u OrClientSemanticTokensRequestOptionsFull) ClientSemanticTokensRequestFullDelta() (ClientSemanticTokensRequestFullDelta, bool) {
	if u.tag != 1 {
		var zero ClientSemanticTokensRequestFullDelta
		return zero, false
	}
	return u.value.(ClientSemanticTokensRequestFullDelta), true
}

func (u OrCompletionItemDocumentation) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrCompletionItemDocumentation) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrCompletionItemDocumentationString(v)
		return nil
	}
	if markupContentMatches(raw) {
		var v MarkupContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrCompletionItemDocumentationMarkupContent(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrCompletionItemDocumentation) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// MarkupContent returns the MarkupContent variant value and true if selected.
func (u OrCompletionItemDocumentation) MarkupContent() (MarkupContent, bool) {
	if u.tag != 1 {
		var zero MarkupContent
		return zero, false
	}
	return u.value.(MarkupContent), true
}

func (u OrCompletionItemTextEdit) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrCompletionItemTextEdit) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if textEditMatches(raw) {
		var v TextEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrCompletionItemTextEditTextEdit(v)
		return nil
	}
	if insertReplaceEditMatches(raw) {
		var v InsertReplaceEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrCompletionItemTextEditInsertReplaceEdit(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// TextEdit returns the TextEdit variant value and true if selected.
func (u OrCompletionItemTextEdit) TextEdit() (TextEdit, bool) {
	if u.tag != 0 {
		var zero TextEdit
		return zero, false
	}
	return u.value.(TextEdit), true
}

// InsertReplaceEdit returns the InsertReplaceEdit variant value and true if selected.
func (u OrCompletionItemTextEdit) InsertReplaceEdit() (InsertReplaceEdit, bool) {
	if u.tag != 1 {
		var zero InsertReplaceEdit
		return zero, false
	}
	return u.value.(InsertReplaceEdit), true
}

func (u OrCompletionItemDefaultsEditRange) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrCompletionItemDefaultsEditRange) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if rangeMatches(raw) {
		var v Range
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrCompletionItemDefaultsEditRangeRange(v)
		return nil
	}
	if editRangeWithInsertReplaceMatches(raw) {
		var v EditRangeWithInsertReplace
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrCompletionItemDefaultsEditRangeEditRangeWithInsertReplace(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Range returns the Range variant value and true if selected.
func (u OrCompletionItemDefaultsEditRange) Range() (Range, bool) {
	if u.tag != 0 {
		var zero Range
		return zero, false
	}
	return u.value.(Range), true
}

// EditRangeWithInsertReplace returns the EditRangeWithInsertReplace variant value and true if selected.
func (u OrCompletionItemDefaultsEditRange) EditRangeWithInsertReplace() (EditRangeWithInsertReplace, bool) {
	if u.tag != 1 {
		var zero EditRangeWithInsertReplace
		return zero, false
	}
	return u.value.(EditRangeWithInsertReplace), true
}

func (u OrDiagnosticCode) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrDiagnosticCode) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isNumber(raw) {
		var v int32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrDiagnosticCodeInteger(v)
		return nil
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrDiagnosticCodeString(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Integer returns the integer variant value and true if selected.
func (u OrDiagnosticCode) Integer() (int32, bool) {
	if u.tag != 0 {
		var zero int32
		return zero, false
	}
	return u.value.(int32), true
}

// String returns the string variant value and true if selected.
func (u OrDiagnosticCode) String() (string, bool) {
	if u.tag != 1 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

func (u OrDiagnosticMessage) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrDiagnosticMessage) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrDiagnosticMessageString(v)
		return nil
	}
	if markupContentMatches(raw) {
		var v MarkupContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrDiagnosticMessageMarkupContent(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrDiagnosticMessage) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// MarkupContent returns the MarkupContent variant value and true if selected.
func (u OrDiagnosticMessage) MarkupContent() (MarkupContent, bool) {
	if u.tag != 1 {
		var zero MarkupContent
		return zero, false
	}
	return u.value.(MarkupContent), true
}

func (u OrDidChangeConfigurationRegistrationOptionsSection) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrDidChangeConfigurationRegistrationOptionsSection) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrDidChangeConfigurationRegistrationOptionsSectionString(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || isString(raw.([]any)[0])) {
		var v []string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrDidChangeConfigurationRegistrationOptionsSectionArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrDidChangeConfigurationRegistrationOptionsSection) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// Array1 returns the string[] variant value and true if selected.
func (u OrDidChangeConfigurationRegistrationOptionsSection) Array1() ([]string, bool) {
	if u.tag != 1 {
		var zero []string
		return zero, false
	}
	return u.value.([]string), true
}

func (u OrHoverContents) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrHoverContents) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if markupContentMatches(raw) {
		var v MarkupContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrHoverContentsMarkupContent(v)
		return nil
	}
	if markedStringMatches(raw) {
		var v MarkedString
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrHoverContentsMarkedString(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || markedStringMatches(raw.([]any)[0])) {
		var v []MarkedString
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrHoverContentsArray2(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// MarkupContent returns the MarkupContent variant value and true if selected.
func (u OrHoverContents) MarkupContent() (MarkupContent, bool) {
	if u.tag != 0 {
		var zero MarkupContent
		return zero, false
	}
	return u.value.(MarkupContent), true
}

// MarkedString returns the MarkedString variant value and true if selected.
func (u OrHoverContents) MarkedString() (MarkedString, bool) {
	if u.tag != 1 {
		var zero MarkedString
		return zero, false
	}
	return u.value.(MarkedString), true
}

// Array2 returns the MarkedString[] variant value and true if selected.
func (u OrHoverContents) Array2() ([]MarkedString, bool) {
	if u.tag != 2 {
		var zero []MarkedString
		return zero, false
	}
	return u.value.([]MarkedString), true
}

func (u OrInlayHintLabel) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrInlayHintLabel) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInlayHintLabelString(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || inlayHintLabelPartMatches(raw.([]any)[0])) {
		var v []InlayHintLabelPart
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInlayHintLabelArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrInlayHintLabel) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// Array1 returns the InlayHintLabelPart[] variant value and true if selected.
func (u OrInlayHintLabel) Array1() ([]InlayHintLabelPart, bool) {
	if u.tag != 1 {
		var zero []InlayHintLabelPart
		return zero, false
	}
	return u.value.([]InlayHintLabelPart), true
}

func (u OrInlayHintTooltip) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrInlayHintTooltip) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInlayHintTooltipString(v)
		return nil
	}
	if markupContentMatches(raw) {
		var v MarkupContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInlayHintTooltipMarkupContent(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrInlayHintTooltip) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// MarkupContent returns the MarkupContent variant value and true if selected.
func (u OrInlayHintTooltip) MarkupContent() (MarkupContent, bool) {
	if u.tag != 1 {
		var zero MarkupContent
		return zero, false
	}
	return u.value.(MarkupContent), true
}

func (u OrInlayHintLabelPartTooltip) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrInlayHintLabelPartTooltip) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInlayHintLabelPartTooltipString(v)
		return nil
	}
	if markupContentMatches(raw) {
		var v MarkupContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInlayHintLabelPartTooltipMarkupContent(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrInlayHintLabelPartTooltip) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// MarkupContent returns the MarkupContent variant value and true if selected.
func (u OrInlayHintLabelPartTooltip) MarkupContent() (MarkupContent, bool) {
	if u.tag != 1 {
		var zero MarkupContent
		return zero, false
	}
	return u.value.(MarkupContent), true
}

func (u OrInlineCompletionItemInsertText) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrInlineCompletionItemInsertText) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInlineCompletionItemInsertTextString(v)
		return nil
	}
	if stringValueMatches(raw) {
		var v StringValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInlineCompletionItemInsertTextStringValue(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrInlineCompletionItemInsertText) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// StringValue returns the StringValue variant value and true if selected.
func (u OrInlineCompletionItemInsertText) StringValue() (StringValue, bool) {
	if u.tag != 1 {
		var zero StringValue
		return zero, false
	}
	return u.value.(StringValue), true
}

func (u OrNotebookCellTextDocumentFilterNotebook) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrNotebookCellTextDocumentFilterNotebook) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrNotebookCellTextDocumentFilterNotebookString(v)
		return nil
	}
	if notebookDocumentFilterMatches(raw) {
		var v NotebookDocumentFilter
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrNotebookCellTextDocumentFilterNotebookNotebookDocumentFilter(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrNotebookCellTextDocumentFilterNotebook) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// NotebookDocumentFilter returns the NotebookDocumentFilter variant value and true if selected.
func (u OrNotebookCellTextDocumentFilterNotebook) NotebookDocumentFilter() (NotebookDocumentFilter, bool) {
	if u.tag != 1 {
		var zero NotebookDocumentFilter
		return zero, false
	}
	return u.value.(NotebookDocumentFilter), true
}

func (u OrNotebookDocumentFilterWithCellsNotebook) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrNotebookDocumentFilterWithCellsNotebook) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrNotebookDocumentFilterWithCellsNotebookString(v)
		return nil
	}
	if notebookDocumentFilterMatches(raw) {
		var v NotebookDocumentFilter
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrNotebookDocumentFilterWithCellsNotebookNotebookDocumentFilter(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrNotebookDocumentFilterWithCellsNotebook) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// NotebookDocumentFilter returns the NotebookDocumentFilter variant value and true if selected.
func (u OrNotebookDocumentFilterWithCellsNotebook) NotebookDocumentFilter() (NotebookDocumentFilter, bool) {
	if u.tag != 1 {
		var zero NotebookDocumentFilter
		return zero, false
	}
	return u.value.(NotebookDocumentFilter), true
}

func (u OrNotebookDocumentFilterWithNotebookNotebook) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrNotebookDocumentFilterWithNotebookNotebook) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrNotebookDocumentFilterWithNotebookNotebookString(v)
		return nil
	}
	if notebookDocumentFilterMatches(raw) {
		var v NotebookDocumentFilter
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrNotebookDocumentFilterWithNotebookNotebookNotebookDocumentFilter(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrNotebookDocumentFilterWithNotebookNotebook) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// NotebookDocumentFilter returns the NotebookDocumentFilter variant value and true if selected.
func (u OrNotebookDocumentFilterWithNotebookNotebook) NotebookDocumentFilter() (NotebookDocumentFilter, bool) {
	if u.tag != 1 {
		var zero NotebookDocumentFilter
		return zero, false
	}
	return u.value.(NotebookDocumentFilter), true
}

func (u OrNotebookDocumentSyncOptionsNotebookSelectorElem) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrNotebookDocumentSyncOptionsNotebookSelectorElem) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if notebookDocumentFilterWithNotebookMatches(raw) {
		var v NotebookDocumentFilterWithNotebook
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrNotebookDocumentSyncOptionsNotebookSelectorElemNotebookDocumentFilterWithNotebook(v)
		return nil
	}
	if notebookDocumentFilterWithCellsMatches(raw) {
		var v NotebookDocumentFilterWithCells
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrNotebookDocumentSyncOptionsNotebookSelectorElemNotebookDocumentFilterWithCells(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// NotebookDocumentFilterWithNotebook returns the NotebookDocumentFilterWithNotebook variant value and true if selected.
func (u OrNotebookDocumentSyncOptionsNotebookSelectorElem) NotebookDocumentFilterWithNotebook() (NotebookDocumentFilterWithNotebook, bool) {
	if u.tag != 0 {
		var zero NotebookDocumentFilterWithNotebook
		return zero, false
	}
	return u.value.(NotebookDocumentFilterWithNotebook), true
}

// NotebookDocumentFilterWithCells returns the NotebookDocumentFilterWithCells variant value and true if selected.
func (u OrNotebookDocumentSyncOptionsNotebookSelectorElem) NotebookDocumentFilterWithCells() (NotebookDocumentFilterWithCells, bool) {
	if u.tag != 1 {
		var zero NotebookDocumentFilterWithCells
		return zero, false
	}
	return u.value.(NotebookDocumentFilterWithCells), true
}

func (u OrOptionalVersionedTextDocumentIdentifierVersion) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrOptionalVersionedTextDocumentIdentifierVersion) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isNumber(raw) {
		var v int32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrOptionalVersionedTextDocumentIdentifierVersionInteger(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Integer returns the integer variant value and true if selected.
func (u OrOptionalVersionedTextDocumentIdentifierVersion) Integer() (int32, bool) {
	if u.tag != 0 {
		var zero int32
		return zero, false
	}
	return u.value.(int32), true
}

func (u OrParameterInformationLabel) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrParameterInformationLabel) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrParameterInformationLabelString(v)
		return nil
	}
	if isArray(raw) && len(raw.([]any)) == 2 && isNumber(raw.([]any)[0]) && isNumber(raw.([]any)[1]) {
		var v TupleParameterInformationLabelItem1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrParameterInformationLabelVariant1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrParameterInformationLabel) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// Variant1 returns the [uinteger, uinteger] variant value and true if selected.
func (u OrParameterInformationLabel) Variant1() (TupleParameterInformationLabelItem1, bool) {
	if u.tag != 1 {
		var zero TupleParameterInformationLabelItem1
		return zero, false
	}
	return u.value.(TupleParameterInformationLabelItem1), true
}

func (u OrParameterInformationDocumentation) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrParameterInformationDocumentation) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrParameterInformationDocumentationString(v)
		return nil
	}
	if markupContentMatches(raw) {
		var v MarkupContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrParameterInformationDocumentationMarkupContent(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrParameterInformationDocumentation) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// MarkupContent returns the MarkupContent variant value and true if selected.
func (u OrParameterInformationDocumentation) MarkupContent() (MarkupContent, bool) {
	if u.tag != 1 {
		var zero MarkupContent
		return zero, false
	}
	return u.value.(MarkupContent), true
}

func (u OrRelativePatternBaseUri) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrRelativePatternBaseUri) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if workspaceFolderMatches(raw) {
		var v WorkspaceFolder
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrRelativePatternBaseUriWorkspaceFolder(v)
		return nil
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrRelativePatternBaseUriURI(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// WorkspaceFolder returns the WorkspaceFolder variant value and true if selected.
func (u OrRelativePatternBaseUri) WorkspaceFolder() (WorkspaceFolder, bool) {
	if u.tag != 0 {
		var zero WorkspaceFolder
		return zero, false
	}
	return u.value.(WorkspaceFolder), true
}

// URI returns the URI variant value and true if selected.
func (u OrRelativePatternBaseUri) URI() (string, bool) {
	if u.tag != 1 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

func (u OrSemanticTokensOptionsRange) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrSemanticTokensOptionsRange) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrSemanticTokensOptionsRangeBoolean(v)
		return nil
	}
	if isObject(raw) {
		var v LitSemanticTokensOptionsRangeItem1
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrSemanticTokensOptionsRangeLiteral1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrSemanticTokensOptionsRange) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// Literal1 returns the literal variant value and true if selected.
func (u OrSemanticTokensOptionsRange) Literal1() (LitSemanticTokensOptionsRangeItem1, bool) {
	if u.tag != 1 {
		var zero LitSemanticTokensOptionsRangeItem1
		return zero, false
	}
	return u.value.(LitSemanticTokensOptionsRangeItem1), true
}

func (u OrSemanticTokensOptionsFull) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrSemanticTokensOptionsFull) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrSemanticTokensOptionsFullBoolean(v)
		return nil
	}
	if semanticTokensFullDeltaMatches(raw) {
		var v SemanticTokensFullDelta
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrSemanticTokensOptionsFullSemanticTokensFullDelta(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrSemanticTokensOptionsFull) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// SemanticTokensFullDelta returns the SemanticTokensFullDelta variant value and true if selected.
func (u OrSemanticTokensOptionsFull) SemanticTokensFullDelta() (SemanticTokensFullDelta, bool) {
	if u.tag != 1 {
		var zero SemanticTokensFullDelta
		return zero, false
	}
	return u.value.(SemanticTokensFullDelta), true
}

func (u OrServerCapabilitiesTextDocumentSync) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesTextDocumentSync) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if textDocumentSyncOptionsMatches(raw) {
		var v TextDocumentSyncOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesTextDocumentSyncTextDocumentSyncOptions(v)
		return nil
	}
	if textDocumentSyncKindMatches(raw) {
		var v TextDocumentSyncKind
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesTextDocumentSyncTextDocumentSyncKind(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// TextDocumentSyncOptions returns the TextDocumentSyncOptions variant value and true if selected.
func (u OrServerCapabilitiesTextDocumentSync) TextDocumentSyncOptions() (TextDocumentSyncOptions, bool) {
	if u.tag != 0 {
		var zero TextDocumentSyncOptions
		return zero, false
	}
	return u.value.(TextDocumentSyncOptions), true
}

// TextDocumentSyncKind returns the TextDocumentSyncKind variant value and true if selected.
func (u OrServerCapabilitiesTextDocumentSync) TextDocumentSyncKind() (TextDocumentSyncKind, bool) {
	if u.tag != 1 {
		var zero TextDocumentSyncKind
		return zero, false
	}
	return u.value.(TextDocumentSyncKind), true
}

func (u OrServerCapabilitiesNotebookDocumentSync) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesNotebookDocumentSync) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if notebookDocumentSyncOptionsMatches(raw) {
		var v NotebookDocumentSyncOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesNotebookDocumentSyncNotebookDocumentSyncOptions(v)
		return nil
	}
	if notebookDocumentSyncRegistrationOptionsMatches(raw) {
		var v NotebookDocumentSyncRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesNotebookDocumentSyncNotebookDocumentSyncRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// NotebookDocumentSyncOptions returns the NotebookDocumentSyncOptions variant value and true if selected.
func (u OrServerCapabilitiesNotebookDocumentSync) NotebookDocumentSyncOptions() (NotebookDocumentSyncOptions, bool) {
	if u.tag != 0 {
		var zero NotebookDocumentSyncOptions
		return zero, false
	}
	return u.value.(NotebookDocumentSyncOptions), true
}

// NotebookDocumentSyncRegistrationOptions returns the NotebookDocumentSyncRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesNotebookDocumentSync) NotebookDocumentSyncRegistrationOptions() (NotebookDocumentSyncRegistrationOptions, bool) {
	if u.tag != 1 {
		var zero NotebookDocumentSyncRegistrationOptions
		return zero, false
	}
	return u.value.(NotebookDocumentSyncRegistrationOptions), true
}

func (u OrServerCapabilitiesHoverProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesHoverProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesHoverProviderBoolean(v)
		return nil
	}
	if hoverOptionsMatches(raw) {
		var v HoverOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesHoverProviderHoverOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesHoverProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// HoverOptions returns the HoverOptions variant value and true if selected.
func (u OrServerCapabilitiesHoverProvider) HoverOptions() (HoverOptions, bool) {
	if u.tag != 1 {
		var zero HoverOptions
		return zero, false
	}
	return u.value.(HoverOptions), true
}

func (u OrServerCapabilitiesDeclarationProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesDeclarationProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDeclarationProviderBoolean(v)
		return nil
	}
	if declarationOptionsMatches(raw) {
		var v DeclarationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDeclarationProviderDeclarationOptions(v)
		return nil
	}
	if declarationRegistrationOptionsMatches(raw) {
		var v DeclarationRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDeclarationProviderDeclarationRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesDeclarationProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// DeclarationOptions returns the DeclarationOptions variant value and true if selected.
func (u OrServerCapabilitiesDeclarationProvider) DeclarationOptions() (DeclarationOptions, bool) {
	if u.tag != 1 {
		var zero DeclarationOptions
		return zero, false
	}
	return u.value.(DeclarationOptions), true
}

// DeclarationRegistrationOptions returns the DeclarationRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesDeclarationProvider) DeclarationRegistrationOptions() (DeclarationRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero DeclarationRegistrationOptions
		return zero, false
	}
	return u.value.(DeclarationRegistrationOptions), true
}

func (u OrServerCapabilitiesDefinitionProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesDefinitionProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDefinitionProviderBoolean(v)
		return nil
	}
	if definitionOptionsMatches(raw) {
		var v DefinitionOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDefinitionProviderDefinitionOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesDefinitionProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// DefinitionOptions returns the DefinitionOptions variant value and true if selected.
func (u OrServerCapabilitiesDefinitionProvider) DefinitionOptions() (DefinitionOptions, bool) {
	if u.tag != 1 {
		var zero DefinitionOptions
		return zero, false
	}
	return u.value.(DefinitionOptions), true
}

func (u OrServerCapabilitiesTypeDefinitionProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesTypeDefinitionProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesTypeDefinitionProviderBoolean(v)
		return nil
	}
	if typeDefinitionOptionsMatches(raw) {
		var v TypeDefinitionOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesTypeDefinitionProviderTypeDefinitionOptions(v)
		return nil
	}
	if typeDefinitionRegistrationOptionsMatches(raw) {
		var v TypeDefinitionRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesTypeDefinitionProviderTypeDefinitionRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesTypeDefinitionProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// TypeDefinitionOptions returns the TypeDefinitionOptions variant value and true if selected.
func (u OrServerCapabilitiesTypeDefinitionProvider) TypeDefinitionOptions() (TypeDefinitionOptions, bool) {
	if u.tag != 1 {
		var zero TypeDefinitionOptions
		return zero, false
	}
	return u.value.(TypeDefinitionOptions), true
}

// TypeDefinitionRegistrationOptions returns the TypeDefinitionRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesTypeDefinitionProvider) TypeDefinitionRegistrationOptions() (TypeDefinitionRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero TypeDefinitionRegistrationOptions
		return zero, false
	}
	return u.value.(TypeDefinitionRegistrationOptions), true
}

func (u OrServerCapabilitiesImplementationProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesImplementationProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesImplementationProviderBoolean(v)
		return nil
	}
	if implementationOptionsMatches(raw) {
		var v ImplementationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesImplementationProviderImplementationOptions(v)
		return nil
	}
	if implementationRegistrationOptionsMatches(raw) {
		var v ImplementationRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesImplementationProviderImplementationRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesImplementationProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// ImplementationOptions returns the ImplementationOptions variant value and true if selected.
func (u OrServerCapabilitiesImplementationProvider) ImplementationOptions() (ImplementationOptions, bool) {
	if u.tag != 1 {
		var zero ImplementationOptions
		return zero, false
	}
	return u.value.(ImplementationOptions), true
}

// ImplementationRegistrationOptions returns the ImplementationRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesImplementationProvider) ImplementationRegistrationOptions() (ImplementationRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero ImplementationRegistrationOptions
		return zero, false
	}
	return u.value.(ImplementationRegistrationOptions), true
}

func (u OrServerCapabilitiesReferencesProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesReferencesProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesReferencesProviderBoolean(v)
		return nil
	}
	if referenceOptionsMatches(raw) {
		var v ReferenceOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesReferencesProviderReferenceOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesReferencesProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// ReferenceOptions returns the ReferenceOptions variant value and true if selected.
func (u OrServerCapabilitiesReferencesProvider) ReferenceOptions() (ReferenceOptions, bool) {
	if u.tag != 1 {
		var zero ReferenceOptions
		return zero, false
	}
	return u.value.(ReferenceOptions), true
}

func (u OrServerCapabilitiesDocumentHighlightProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesDocumentHighlightProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDocumentHighlightProviderBoolean(v)
		return nil
	}
	if documentHighlightOptionsMatches(raw) {
		var v DocumentHighlightOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDocumentHighlightProviderDocumentHighlightOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesDocumentHighlightProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// DocumentHighlightOptions returns the DocumentHighlightOptions variant value and true if selected.
func (u OrServerCapabilitiesDocumentHighlightProvider) DocumentHighlightOptions() (DocumentHighlightOptions, bool) {
	if u.tag != 1 {
		var zero DocumentHighlightOptions
		return zero, false
	}
	return u.value.(DocumentHighlightOptions), true
}

func (u OrServerCapabilitiesDocumentSymbolProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesDocumentSymbolProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDocumentSymbolProviderBoolean(v)
		return nil
	}
	if documentSymbolOptionsMatches(raw) {
		var v DocumentSymbolOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDocumentSymbolProviderDocumentSymbolOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesDocumentSymbolProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// DocumentSymbolOptions returns the DocumentSymbolOptions variant value and true if selected.
func (u OrServerCapabilitiesDocumentSymbolProvider) DocumentSymbolOptions() (DocumentSymbolOptions, bool) {
	if u.tag != 1 {
		var zero DocumentSymbolOptions
		return zero, false
	}
	return u.value.(DocumentSymbolOptions), true
}

func (u OrServerCapabilitiesCodeActionProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesCodeActionProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesCodeActionProviderBoolean(v)
		return nil
	}
	if codeActionOptionsMatches(raw) {
		var v CodeActionOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesCodeActionProviderCodeActionOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesCodeActionProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// CodeActionOptions returns the CodeActionOptions variant value and true if selected.
func (u OrServerCapabilitiesCodeActionProvider) CodeActionOptions() (CodeActionOptions, bool) {
	if u.tag != 1 {
		var zero CodeActionOptions
		return zero, false
	}
	return u.value.(CodeActionOptions), true
}

func (u OrServerCapabilitiesColorProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesColorProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesColorProviderBoolean(v)
		return nil
	}
	if documentColorOptionsMatches(raw) {
		var v DocumentColorOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesColorProviderDocumentColorOptions(v)
		return nil
	}
	if documentColorRegistrationOptionsMatches(raw) {
		var v DocumentColorRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesColorProviderDocumentColorRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesColorProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// DocumentColorOptions returns the DocumentColorOptions variant value and true if selected.
func (u OrServerCapabilitiesColorProvider) DocumentColorOptions() (DocumentColorOptions, bool) {
	if u.tag != 1 {
		var zero DocumentColorOptions
		return zero, false
	}
	return u.value.(DocumentColorOptions), true
}

// DocumentColorRegistrationOptions returns the DocumentColorRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesColorProvider) DocumentColorRegistrationOptions() (DocumentColorRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero DocumentColorRegistrationOptions
		return zero, false
	}
	return u.value.(DocumentColorRegistrationOptions), true
}

func (u OrServerCapabilitiesWorkspaceSymbolProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesWorkspaceSymbolProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesWorkspaceSymbolProviderBoolean(v)
		return nil
	}
	if workspaceSymbolOptionsMatches(raw) {
		var v WorkspaceSymbolOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesWorkspaceSymbolProviderWorkspaceSymbolOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesWorkspaceSymbolProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// WorkspaceSymbolOptions returns the WorkspaceSymbolOptions variant value and true if selected.
func (u OrServerCapabilitiesWorkspaceSymbolProvider) WorkspaceSymbolOptions() (WorkspaceSymbolOptions, bool) {
	if u.tag != 1 {
		var zero WorkspaceSymbolOptions
		return zero, false
	}
	return u.value.(WorkspaceSymbolOptions), true
}

func (u OrServerCapabilitiesDocumentFormattingProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesDocumentFormattingProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDocumentFormattingProviderBoolean(v)
		return nil
	}
	if documentFormattingOptionsMatches(raw) {
		var v DocumentFormattingOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDocumentFormattingProviderDocumentFormattingOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesDocumentFormattingProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// DocumentFormattingOptions returns the DocumentFormattingOptions variant value and true if selected.
func (u OrServerCapabilitiesDocumentFormattingProvider) DocumentFormattingOptions() (DocumentFormattingOptions, bool) {
	if u.tag != 1 {
		var zero DocumentFormattingOptions
		return zero, false
	}
	return u.value.(DocumentFormattingOptions), true
}

func (u OrServerCapabilitiesDocumentRangeFormattingProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesDocumentRangeFormattingProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDocumentRangeFormattingProviderBoolean(v)
		return nil
	}
	if documentRangeFormattingOptionsMatches(raw) {
		var v DocumentRangeFormattingOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDocumentRangeFormattingProviderDocumentRangeFormattingOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesDocumentRangeFormattingProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// DocumentRangeFormattingOptions returns the DocumentRangeFormattingOptions variant value and true if selected.
func (u OrServerCapabilitiesDocumentRangeFormattingProvider) DocumentRangeFormattingOptions() (DocumentRangeFormattingOptions, bool) {
	if u.tag != 1 {
		var zero DocumentRangeFormattingOptions
		return zero, false
	}
	return u.value.(DocumentRangeFormattingOptions), true
}

func (u OrServerCapabilitiesRenameProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesRenameProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesRenameProviderBoolean(v)
		return nil
	}
	if renameOptionsMatches(raw) {
		var v RenameOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesRenameProviderRenameOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesRenameProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// RenameOptions returns the RenameOptions variant value and true if selected.
func (u OrServerCapabilitiesRenameProvider) RenameOptions() (RenameOptions, bool) {
	if u.tag != 1 {
		var zero RenameOptions
		return zero, false
	}
	return u.value.(RenameOptions), true
}

func (u OrServerCapabilitiesFoldingRangeProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesFoldingRangeProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesFoldingRangeProviderBoolean(v)
		return nil
	}
	if foldingRangeOptionsMatches(raw) {
		var v FoldingRangeOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesFoldingRangeProviderFoldingRangeOptions(v)
		return nil
	}
	if foldingRangeRegistrationOptionsMatches(raw) {
		var v FoldingRangeRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesFoldingRangeProviderFoldingRangeRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesFoldingRangeProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// FoldingRangeOptions returns the FoldingRangeOptions variant value and true if selected.
func (u OrServerCapabilitiesFoldingRangeProvider) FoldingRangeOptions() (FoldingRangeOptions, bool) {
	if u.tag != 1 {
		var zero FoldingRangeOptions
		return zero, false
	}
	return u.value.(FoldingRangeOptions), true
}

// FoldingRangeRegistrationOptions returns the FoldingRangeRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesFoldingRangeProvider) FoldingRangeRegistrationOptions() (FoldingRangeRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero FoldingRangeRegistrationOptions
		return zero, false
	}
	return u.value.(FoldingRangeRegistrationOptions), true
}

func (u OrServerCapabilitiesSelectionRangeProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesSelectionRangeProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesSelectionRangeProviderBoolean(v)
		return nil
	}
	if selectionRangeOptionsMatches(raw) {
		var v SelectionRangeOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesSelectionRangeProviderSelectionRangeOptions(v)
		return nil
	}
	if selectionRangeRegistrationOptionsMatches(raw) {
		var v SelectionRangeRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesSelectionRangeProviderSelectionRangeRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesSelectionRangeProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// SelectionRangeOptions returns the SelectionRangeOptions variant value and true if selected.
func (u OrServerCapabilitiesSelectionRangeProvider) SelectionRangeOptions() (SelectionRangeOptions, bool) {
	if u.tag != 1 {
		var zero SelectionRangeOptions
		return zero, false
	}
	return u.value.(SelectionRangeOptions), true
}

// SelectionRangeRegistrationOptions returns the SelectionRangeRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesSelectionRangeProvider) SelectionRangeRegistrationOptions() (SelectionRangeRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero SelectionRangeRegistrationOptions
		return zero, false
	}
	return u.value.(SelectionRangeRegistrationOptions), true
}

func (u OrServerCapabilitiesCallHierarchyProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesCallHierarchyProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesCallHierarchyProviderBoolean(v)
		return nil
	}
	if callHierarchyOptionsMatches(raw) {
		var v CallHierarchyOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesCallHierarchyProviderCallHierarchyOptions(v)
		return nil
	}
	if callHierarchyRegistrationOptionsMatches(raw) {
		var v CallHierarchyRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesCallHierarchyProviderCallHierarchyRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesCallHierarchyProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// CallHierarchyOptions returns the CallHierarchyOptions variant value and true if selected.
func (u OrServerCapabilitiesCallHierarchyProvider) CallHierarchyOptions() (CallHierarchyOptions, bool) {
	if u.tag != 1 {
		var zero CallHierarchyOptions
		return zero, false
	}
	return u.value.(CallHierarchyOptions), true
}

// CallHierarchyRegistrationOptions returns the CallHierarchyRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesCallHierarchyProvider) CallHierarchyRegistrationOptions() (CallHierarchyRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero CallHierarchyRegistrationOptions
		return zero, false
	}
	return u.value.(CallHierarchyRegistrationOptions), true
}

func (u OrServerCapabilitiesLinkedEditingRangeProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesLinkedEditingRangeProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesLinkedEditingRangeProviderBoolean(v)
		return nil
	}
	if linkedEditingRangeOptionsMatches(raw) {
		var v LinkedEditingRangeOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesLinkedEditingRangeProviderLinkedEditingRangeOptions(v)
		return nil
	}
	if linkedEditingRangeRegistrationOptionsMatches(raw) {
		var v LinkedEditingRangeRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesLinkedEditingRangeProviderLinkedEditingRangeRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesLinkedEditingRangeProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// LinkedEditingRangeOptions returns the LinkedEditingRangeOptions variant value and true if selected.
func (u OrServerCapabilitiesLinkedEditingRangeProvider) LinkedEditingRangeOptions() (LinkedEditingRangeOptions, bool) {
	if u.tag != 1 {
		var zero LinkedEditingRangeOptions
		return zero, false
	}
	return u.value.(LinkedEditingRangeOptions), true
}

// LinkedEditingRangeRegistrationOptions returns the LinkedEditingRangeRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesLinkedEditingRangeProvider) LinkedEditingRangeRegistrationOptions() (LinkedEditingRangeRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero LinkedEditingRangeRegistrationOptions
		return zero, false
	}
	return u.value.(LinkedEditingRangeRegistrationOptions), true
}

func (u OrServerCapabilitiesSemanticTokensProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesSemanticTokensProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if semanticTokensOptionsMatches(raw) {
		var v SemanticTokensOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesSemanticTokensProviderSemanticTokensOptions(v)
		return nil
	}
	if semanticTokensRegistrationOptionsMatches(raw) {
		var v SemanticTokensRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesSemanticTokensProviderSemanticTokensRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// SemanticTokensOptions returns the SemanticTokensOptions variant value and true if selected.
func (u OrServerCapabilitiesSemanticTokensProvider) SemanticTokensOptions() (SemanticTokensOptions, bool) {
	if u.tag != 0 {
		var zero SemanticTokensOptions
		return zero, false
	}
	return u.value.(SemanticTokensOptions), true
}

// SemanticTokensRegistrationOptions returns the SemanticTokensRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesSemanticTokensProvider) SemanticTokensRegistrationOptions() (SemanticTokensRegistrationOptions, bool) {
	if u.tag != 1 {
		var zero SemanticTokensRegistrationOptions
		return zero, false
	}
	return u.value.(SemanticTokensRegistrationOptions), true
}

func (u OrServerCapabilitiesMonikerProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesMonikerProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesMonikerProviderBoolean(v)
		return nil
	}
	if monikerOptionsMatches(raw) {
		var v MonikerOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesMonikerProviderMonikerOptions(v)
		return nil
	}
	if monikerRegistrationOptionsMatches(raw) {
		var v MonikerRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesMonikerProviderMonikerRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesMonikerProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// MonikerOptions returns the MonikerOptions variant value and true if selected.
func (u OrServerCapabilitiesMonikerProvider) MonikerOptions() (MonikerOptions, bool) {
	if u.tag != 1 {
		var zero MonikerOptions
		return zero, false
	}
	return u.value.(MonikerOptions), true
}

// MonikerRegistrationOptions returns the MonikerRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesMonikerProvider) MonikerRegistrationOptions() (MonikerRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero MonikerRegistrationOptions
		return zero, false
	}
	return u.value.(MonikerRegistrationOptions), true
}

func (u OrServerCapabilitiesTypeHierarchyProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesTypeHierarchyProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesTypeHierarchyProviderBoolean(v)
		return nil
	}
	if typeHierarchyOptionsMatches(raw) {
		var v TypeHierarchyOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesTypeHierarchyProviderTypeHierarchyOptions(v)
		return nil
	}
	if typeHierarchyRegistrationOptionsMatches(raw) {
		var v TypeHierarchyRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesTypeHierarchyProviderTypeHierarchyRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesTypeHierarchyProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// TypeHierarchyOptions returns the TypeHierarchyOptions variant value and true if selected.
func (u OrServerCapabilitiesTypeHierarchyProvider) TypeHierarchyOptions() (TypeHierarchyOptions, bool) {
	if u.tag != 1 {
		var zero TypeHierarchyOptions
		return zero, false
	}
	return u.value.(TypeHierarchyOptions), true
}

// TypeHierarchyRegistrationOptions returns the TypeHierarchyRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesTypeHierarchyProvider) TypeHierarchyRegistrationOptions() (TypeHierarchyRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero TypeHierarchyRegistrationOptions
		return zero, false
	}
	return u.value.(TypeHierarchyRegistrationOptions), true
}

func (u OrServerCapabilitiesInlineValueProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesInlineValueProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesInlineValueProviderBoolean(v)
		return nil
	}
	if inlineValueOptionsMatches(raw) {
		var v InlineValueOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesInlineValueProviderInlineValueOptions(v)
		return nil
	}
	if inlineValueRegistrationOptionsMatches(raw) {
		var v InlineValueRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesInlineValueProviderInlineValueRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesInlineValueProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// InlineValueOptions returns the InlineValueOptions variant value and true if selected.
func (u OrServerCapabilitiesInlineValueProvider) InlineValueOptions() (InlineValueOptions, bool) {
	if u.tag != 1 {
		var zero InlineValueOptions
		return zero, false
	}
	return u.value.(InlineValueOptions), true
}

// InlineValueRegistrationOptions returns the InlineValueRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesInlineValueProvider) InlineValueRegistrationOptions() (InlineValueRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero InlineValueRegistrationOptions
		return zero, false
	}
	return u.value.(InlineValueRegistrationOptions), true
}

func (u OrServerCapabilitiesInlayHintProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesInlayHintProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesInlayHintProviderBoolean(v)
		return nil
	}
	if inlayHintOptionsMatches(raw) {
		var v InlayHintOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesInlayHintProviderInlayHintOptions(v)
		return nil
	}
	if inlayHintRegistrationOptionsMatches(raw) {
		var v InlayHintRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesInlayHintProviderInlayHintRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesInlayHintProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// InlayHintOptions returns the InlayHintOptions variant value and true if selected.
func (u OrServerCapabilitiesInlayHintProvider) InlayHintOptions() (InlayHintOptions, bool) {
	if u.tag != 1 {
		var zero InlayHintOptions
		return zero, false
	}
	return u.value.(InlayHintOptions), true
}

// InlayHintRegistrationOptions returns the InlayHintRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesInlayHintProvider) InlayHintRegistrationOptions() (InlayHintRegistrationOptions, bool) {
	if u.tag != 2 {
		var zero InlayHintRegistrationOptions
		return zero, false
	}
	return u.value.(InlayHintRegistrationOptions), true
}

func (u OrServerCapabilitiesDiagnosticProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesDiagnosticProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if diagnosticOptionsMatches(raw) {
		var v DiagnosticOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDiagnosticProviderDiagnosticOptions(v)
		return nil
	}
	if diagnosticRegistrationOptionsMatches(raw) {
		var v DiagnosticRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesDiagnosticProviderDiagnosticRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// DiagnosticOptions returns the DiagnosticOptions variant value and true if selected.
func (u OrServerCapabilitiesDiagnosticProvider) DiagnosticOptions() (DiagnosticOptions, bool) {
	if u.tag != 0 {
		var zero DiagnosticOptions
		return zero, false
	}
	return u.value.(DiagnosticOptions), true
}

// DiagnosticRegistrationOptions returns the DiagnosticRegistrationOptions variant value and true if selected.
func (u OrServerCapabilitiesDiagnosticProvider) DiagnosticRegistrationOptions() (DiagnosticRegistrationOptions, bool) {
	if u.tag != 1 {
		var zero DiagnosticRegistrationOptions
		return zero, false
	}
	return u.value.(DiagnosticRegistrationOptions), true
}

func (u OrServerCapabilitiesInlineCompletionProvider) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrServerCapabilitiesInlineCompletionProvider) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesInlineCompletionProviderBoolean(v)
		return nil
	}
	if inlineCompletionOptionsMatches(raw) {
		var v InlineCompletionOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrServerCapabilitiesInlineCompletionProviderInlineCompletionOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrServerCapabilitiesInlineCompletionProvider) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// InlineCompletionOptions returns the InlineCompletionOptions variant value and true if selected.
func (u OrServerCapabilitiesInlineCompletionProvider) InlineCompletionOptions() (InlineCompletionOptions, bool) {
	if u.tag != 1 {
		var zero InlineCompletionOptions
		return zero, false
	}
	return u.value.(InlineCompletionOptions), true
}

func (u OrSignatureHelpActiveParameter) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrSignatureHelpActiveParameter) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isNumber(raw) {
		var v uint32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrSignatureHelpActiveParameterUinteger(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Uinteger returns the uinteger variant value and true if selected.
func (u OrSignatureHelpActiveParameter) Uinteger() (uint32, bool) {
	if u.tag != 0 {
		var zero uint32
		return zero, false
	}
	return u.value.(uint32), true
}

func (u OrSignatureInformationDocumentation) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrSignatureInformationDocumentation) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrSignatureInformationDocumentationString(v)
		return nil
	}
	if markupContentMatches(raw) {
		var v MarkupContent
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrSignatureInformationDocumentationMarkupContent(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrSignatureInformationDocumentation) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// MarkupContent returns the MarkupContent variant value and true if selected.
func (u OrSignatureInformationDocumentation) MarkupContent() (MarkupContent, bool) {
	if u.tag != 1 {
		var zero MarkupContent
		return zero, false
	}
	return u.value.(MarkupContent), true
}

func (u OrSignatureInformationActiveParameter) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrSignatureInformationActiveParameter) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isNumber(raw) {
		var v uint32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrSignatureInformationActiveParameterUinteger(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Uinteger returns the uinteger variant value and true if selected.
func (u OrSignatureInformationActiveParameter) Uinteger() (uint32, bool) {
	if u.tag != 0 {
		var zero uint32
		return zero, false
	}
	return u.value.(uint32), true
}

func (u OrTextDocumentEditEditsElem) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrTextDocumentEditEditsElem) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if textEditMatches(raw) {
		var v TextEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrTextDocumentEditEditsElemTextEdit(v)
		return nil
	}
	if annotatedTextEditMatches(raw) {
		var v AnnotatedTextEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrTextDocumentEditEditsElemAnnotatedTextEdit(v)
		return nil
	}
	if snippetTextEditMatches(raw) {
		var v SnippetTextEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrTextDocumentEditEditsElemSnippetTextEdit(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// TextEdit returns the TextEdit variant value and true if selected.
func (u OrTextDocumentEditEditsElem) TextEdit() (TextEdit, bool) {
	if u.tag != 0 {
		var zero TextEdit
		return zero, false
	}
	return u.value.(TextEdit), true
}

// AnnotatedTextEdit returns the AnnotatedTextEdit variant value and true if selected.
func (u OrTextDocumentEditEditsElem) AnnotatedTextEdit() (AnnotatedTextEdit, bool) {
	if u.tag != 1 {
		var zero AnnotatedTextEdit
		return zero, false
	}
	return u.value.(AnnotatedTextEdit), true
}

// SnippetTextEdit returns the SnippetTextEdit variant value and true if selected.
func (u OrTextDocumentEditEditsElem) SnippetTextEdit() (SnippetTextEdit, bool) {
	if u.tag != 2 {
		var zero SnippetTextEdit
		return zero, false
	}
	return u.value.(SnippetTextEdit), true
}

func (u OrTextDocumentRegistrationOptionsDocumentSelector) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrTextDocumentRegistrationOptionsDocumentSelector) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if documentSelectorMatches(raw) {
		var v DocumentSelector
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrTextDocumentRegistrationOptionsDocumentSelectorDocumentSelector(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// DocumentSelector returns the DocumentSelector variant value and true if selected.
func (u OrTextDocumentRegistrationOptionsDocumentSelector) DocumentSelector() (DocumentSelector, bool) {
	if u.tag != 0 {
		var zero DocumentSelector
		return zero, false
	}
	return u.value.(DocumentSelector), true
}

func (u OrTextDocumentSyncOptionsSave) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrTextDocumentSyncOptionsSave) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrTextDocumentSyncOptionsSaveBoolean(v)
		return nil
	}
	if saveOptionsMatches(raw) {
		var v SaveOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrTextDocumentSyncOptionsSaveSaveOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Boolean returns the boolean variant value and true if selected.
func (u OrTextDocumentSyncOptionsSave) Boolean() (bool, bool) {
	if u.tag != 0 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

// SaveOptions returns the SaveOptions variant value and true if selected.
func (u OrTextDocumentSyncOptionsSave) SaveOptions() (SaveOptions, bool) {
	if u.tag != 1 {
		var zero SaveOptions
		return zero, false
	}
	return u.value.(SaveOptions), true
}

func (u OrWorkspaceEditDocumentChangesElem) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	case 2:
		return json.Marshal(u.value)
	case 3:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrWorkspaceEditDocumentChangesElem) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if textDocumentEditMatches(raw) {
		var v TextDocumentEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceEditDocumentChangesElemTextDocumentEdit(v)
		return nil
	}
	if createFileMatches(raw) {
		var v CreateFile
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceEditDocumentChangesElemCreateFile(v)
		return nil
	}
	if renameFileMatches(raw) {
		var v RenameFile
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceEditDocumentChangesElemRenameFile(v)
		return nil
	}
	if deleteFileMatches(raw) {
		var v DeleteFile
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceEditDocumentChangesElemDeleteFile(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// TextDocumentEdit returns the TextDocumentEdit variant value and true if selected.
func (u OrWorkspaceEditDocumentChangesElem) TextDocumentEdit() (TextDocumentEdit, bool) {
	if u.tag != 0 {
		var zero TextDocumentEdit
		return zero, false
	}
	return u.value.(TextDocumentEdit), true
}

// CreateFile returns the CreateFile variant value and true if selected.
func (u OrWorkspaceEditDocumentChangesElem) CreateFile() (CreateFile, bool) {
	if u.tag != 1 {
		var zero CreateFile
		return zero, false
	}
	return u.value.(CreateFile), true
}

// RenameFile returns the RenameFile variant value and true if selected.
func (u OrWorkspaceEditDocumentChangesElem) RenameFile() (RenameFile, bool) {
	if u.tag != 2 {
		var zero RenameFile
		return zero, false
	}
	return u.value.(RenameFile), true
}

// DeleteFile returns the DeleteFile variant value and true if selected.
func (u OrWorkspaceEditDocumentChangesElem) DeleteFile() (DeleteFile, bool) {
	if u.tag != 3 {
		var zero DeleteFile
		return zero, false
	}
	return u.value.(DeleteFile), true
}

func (u OrWorkspaceFoldersInitializeParamsWorkspaceFolders) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrWorkspaceFoldersInitializeParamsWorkspaceFolders) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || workspaceFolderMatches(raw.([]any)[0])) {
		var v []WorkspaceFolder
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceFoldersInitializeParamsWorkspaceFoldersArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the WorkspaceFolder[] variant value and true if selected.
func (u OrWorkspaceFoldersInitializeParamsWorkspaceFolders) Array0() ([]WorkspaceFolder, bool) {
	if u.tag != 0 {
		var zero []WorkspaceFolder
		return zero, false
	}
	return u.value.([]WorkspaceFolder), true
}

func (u OrWorkspaceFoldersServerCapabilitiesChangeNotifications) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrWorkspaceFoldersServerCapabilitiesChangeNotifications) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceFoldersServerCapabilitiesChangeNotificationsString(v)
		return nil
	}
	if isBool(raw) {
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceFoldersServerCapabilitiesChangeNotificationsBoolean(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrWorkspaceFoldersServerCapabilitiesChangeNotifications) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

// Boolean returns the boolean variant value and true if selected.
func (u OrWorkspaceFoldersServerCapabilitiesChangeNotifications) Boolean() (bool, bool) {
	if u.tag != 1 {
		var zero bool
		return zero, false
	}
	return u.value.(bool), true
}

func (u OrWorkspaceFullDocumentDiagnosticReportVersion) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrWorkspaceFullDocumentDiagnosticReportVersion) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isNumber(raw) {
		var v int32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceFullDocumentDiagnosticReportVersionInteger(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Integer returns the integer variant value and true if selected.
func (u OrWorkspaceFullDocumentDiagnosticReportVersion) Integer() (int32, bool) {
	if u.tag != 0 {
		var zero int32
		return zero, false
	}
	return u.value.(int32), true
}

func (u OrWorkspaceOptionsTextDocumentContent) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrWorkspaceOptionsTextDocumentContent) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if textDocumentContentOptionsMatches(raw) {
		var v TextDocumentContentOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceOptionsTextDocumentContentTextDocumentContentOptions(v)
		return nil
	}
	if textDocumentContentRegistrationOptionsMatches(raw) {
		var v TextDocumentContentRegistrationOptions
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceOptionsTextDocumentContentTextDocumentContentRegistrationOptions(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// TextDocumentContentOptions returns the TextDocumentContentOptions variant value and true if selected.
func (u OrWorkspaceOptionsTextDocumentContent) TextDocumentContentOptions() (TextDocumentContentOptions, bool) {
	if u.tag != 0 {
		var zero TextDocumentContentOptions
		return zero, false
	}
	return u.value.(TextDocumentContentOptions), true
}

// TextDocumentContentRegistrationOptions returns the TextDocumentContentRegistrationOptions variant value and true if selected.
func (u OrWorkspaceOptionsTextDocumentContent) TextDocumentContentRegistrationOptions() (TextDocumentContentRegistrationOptions, bool) {
	if u.tag != 1 {
		var zero TextDocumentContentRegistrationOptions
		return zero, false
	}
	return u.value.(TextDocumentContentRegistrationOptions), true
}

func (u OrWorkspaceSymbolLocation) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrWorkspaceSymbolLocation) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if locationMatches(raw) {
		var v Location
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceSymbolLocationLocation(v)
		return nil
	}
	if locationUriOnlyMatches(raw) {
		var v LocationUriOnly
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceSymbolLocationLocationUriOnly(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Location returns the Location variant value and true if selected.
func (u OrWorkspaceSymbolLocation) Location() (Location, bool) {
	if u.tag != 0 {
		var zero Location
		return zero, false
	}
	return u.value.(Location), true
}

// LocationUriOnly returns the LocationUriOnly variant value and true if selected.
func (u OrWorkspaceSymbolLocation) LocationUriOnly() (LocationUriOnly, bool) {
	if u.tag != 1 {
		var zero LocationUriOnly
		return zero, false
	}
	return u.value.(LocationUriOnly), true
}

func (u OrWorkspaceUnchangedDocumentDiagnosticReportVersion) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrWorkspaceUnchangedDocumentDiagnosticReportVersion) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isNumber(raw) {
		var v int32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrWorkspaceUnchangedDocumentDiagnosticReportVersionInteger(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Integer returns the integer variant value and true if selected.
func (u OrWorkspaceUnchangedDocumentDiagnosticReportVersion) Integer() (int32, bool) {
	if u.tag != 0 {
		var zero int32
		return zero, false
	}
	return u.value.(int32), true
}

func (u OrInitializeParamsProcessId) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrInitializeParamsProcessId) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isNumber(raw) {
		var v int32
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInitializeParamsProcessIdInteger(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Integer returns the integer variant value and true if selected.
func (u OrInitializeParamsProcessId) Integer() (int32, bool) {
	if u.tag != 0 {
		var zero int32
		return zero, false
	}
	return u.value.(int32), true
}

func (u OrInitializeParamsRootPath) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrInitializeParamsRootPath) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v string
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInitializeParamsRootPathString(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// String returns the string variant value and true if selected.
func (u OrInitializeParamsRootPath) String() (string, bool) {
	if u.tag != 0 {
		var zero string
		return zero, false
	}
	return u.value.(string), true
}

func (u OrInitializeParamsRootUri) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrInitializeParamsRootUri) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isString(raw) {
		var v DocumentURI
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrInitializeParamsRootUriDocumentURI(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// DocumentURI returns the DocumentUri variant value and true if selected.
func (u OrInitializeParamsRootUri) DocumentURI() (DocumentURI, bool) {
	if u.tag != 0 {
		var zero DocumentURI
		return zero, false
	}
	return u.value.(DocumentURI), true
}

func (u OrResultCallHierarchyIncomingCalls) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultCallHierarchyIncomingCalls) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || callHierarchyIncomingCallMatches(raw.([]any)[0])) {
		var v []CallHierarchyIncomingCall
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultCallHierarchyIncomingCallsArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the CallHierarchyIncomingCall[] variant value and true if selected.
func (u OrResultCallHierarchyIncomingCalls) Array0() ([]CallHierarchyIncomingCall, bool) {
	if u.tag != 0 {
		var zero []CallHierarchyIncomingCall
		return zero, false
	}
	return u.value.([]CallHierarchyIncomingCall), true
}

func (u OrResultCallHierarchyOutgoingCalls) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultCallHierarchyOutgoingCalls) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || callHierarchyOutgoingCallMatches(raw.([]any)[0])) {
		var v []CallHierarchyOutgoingCall
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultCallHierarchyOutgoingCallsArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the CallHierarchyOutgoingCall[] variant value and true if selected.
func (u OrResultCallHierarchyOutgoingCalls) Array0() ([]CallHierarchyOutgoingCall, bool) {
	if u.tag != 0 {
		var zero []CallHierarchyOutgoingCall
		return zero, false
	}
	return u.value.([]CallHierarchyOutgoingCall), true
}

func (u OrResultTextDocumentCodeAction) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentCodeAction) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || true) {
		var v []OrResultTextDocumentCodeActionItem0Elem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentCodeActionArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the Command | CodeAction[] variant value and true if selected.
func (u OrResultTextDocumentCodeAction) Array0() ([]OrResultTextDocumentCodeActionItem0Elem, bool) {
	if u.tag != 0 {
		var zero []OrResultTextDocumentCodeActionItem0Elem
		return zero, false
	}
	return u.value.([]OrResultTextDocumentCodeActionItem0Elem), true
}

func (u OrResultTextDocumentCodeActionItem0Elem) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentCodeActionItem0Elem) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if commandMatches(raw) {
		var v Command
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentCodeActionItem0ElemCommand(v)
		return nil
	}
	if codeActionMatches(raw) {
		var v CodeAction
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentCodeActionItem0ElemCodeAction(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Command returns the Command variant value and true if selected.
func (u OrResultTextDocumentCodeActionItem0Elem) Command() (Command, bool) {
	if u.tag != 0 {
		var zero Command
		return zero, false
	}
	return u.value.(Command), true
}

// CodeAction returns the CodeAction variant value and true if selected.
func (u OrResultTextDocumentCodeActionItem0Elem) CodeAction() (CodeAction, bool) {
	if u.tag != 1 {
		var zero CodeAction
		return zero, false
	}
	return u.value.(CodeAction), true
}

func (u OrResultTextDocumentCodeLens) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentCodeLens) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || codeLensMatches(raw.([]any)[0])) {
		var v []CodeLens
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentCodeLensArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the CodeLens[] variant value and true if selected.
func (u OrResultTextDocumentCodeLens) Array0() ([]CodeLens, bool) {
	if u.tag != 0 {
		var zero []CodeLens
		return zero, false
	}
	return u.value.([]CodeLens), true
}

func (u OrResultTextDocumentCompletion) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentCompletion) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || completionItemMatches(raw.([]any)[0])) {
		var v []CompletionItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentCompletionArray0(v)
		return nil
	}
	if completionListMatches(raw) {
		var v CompletionList
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentCompletionCompletionList(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the CompletionItem[] variant value and true if selected.
func (u OrResultTextDocumentCompletion) Array0() ([]CompletionItem, bool) {
	if u.tag != 0 {
		var zero []CompletionItem
		return zero, false
	}
	return u.value.([]CompletionItem), true
}

// CompletionList returns the CompletionList variant value and true if selected.
func (u OrResultTextDocumentCompletion) CompletionList() (CompletionList, bool) {
	if u.tag != 1 {
		var zero CompletionList
		return zero, false
	}
	return u.value.(CompletionList), true
}

func (u OrResultTextDocumentDeclaration) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentDeclaration) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if declarationMatches(raw) {
		var v Declaration
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentDeclarationDeclaration(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || declarationLinkMatches(raw.([]any)[0])) {
		var v []DeclarationLink
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentDeclarationArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Declaration returns the Declaration variant value and true if selected.
func (u OrResultTextDocumentDeclaration) Declaration() (Declaration, bool) {
	if u.tag != 0 {
		var zero Declaration
		return zero, false
	}
	return u.value.(Declaration), true
}

// Array1 returns the DeclarationLink[] variant value and true if selected.
func (u OrResultTextDocumentDeclaration) Array1() ([]DeclarationLink, bool) {
	if u.tag != 1 {
		var zero []DeclarationLink
		return zero, false
	}
	return u.value.([]DeclarationLink), true
}

func (u OrResultTextDocumentDefinition) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentDefinition) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if definitionMatches(raw) {
		var v Definition
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentDefinitionDefinition(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || definitionLinkMatches(raw.([]any)[0])) {
		var v []DefinitionLink
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentDefinitionArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Definition returns the Definition variant value and true if selected.
func (u OrResultTextDocumentDefinition) Definition() (Definition, bool) {
	if u.tag != 0 {
		var zero Definition
		return zero, false
	}
	return u.value.(Definition), true
}

// Array1 returns the DefinitionLink[] variant value and true if selected.
func (u OrResultTextDocumentDefinition) Array1() ([]DefinitionLink, bool) {
	if u.tag != 1 {
		var zero []DefinitionLink
		return zero, false
	}
	return u.value.([]DefinitionLink), true
}

func (u OrResultTextDocumentDocumentHighlight) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentDocumentHighlight) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || documentHighlightMatches(raw.([]any)[0])) {
		var v []DocumentHighlight
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentDocumentHighlightArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the DocumentHighlight[] variant value and true if selected.
func (u OrResultTextDocumentDocumentHighlight) Array0() ([]DocumentHighlight, bool) {
	if u.tag != 0 {
		var zero []DocumentHighlight
		return zero, false
	}
	return u.value.([]DocumentHighlight), true
}

func (u OrResultTextDocumentDocumentLink) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentDocumentLink) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || documentLinkMatches(raw.([]any)[0])) {
		var v []DocumentLink
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentDocumentLinkArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the DocumentLink[] variant value and true if selected.
func (u OrResultTextDocumentDocumentLink) Array0() ([]DocumentLink, bool) {
	if u.tag != 0 {
		var zero []DocumentLink
		return zero, false
	}
	return u.value.([]DocumentLink), true
}

func (u OrResultTextDocumentDocumentSymbol) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentDocumentSymbol) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || symbolInformationMatches(raw.([]any)[0])) {
		var v []SymbolInformation
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentDocumentSymbolArray0(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || documentSymbolMatches(raw.([]any)[0])) {
		var v []DocumentSymbol
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentDocumentSymbolArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the SymbolInformation[] variant value and true if selected.
func (u OrResultTextDocumentDocumentSymbol) Array0() ([]SymbolInformation, bool) {
	if u.tag != 0 {
		var zero []SymbolInformation
		return zero, false
	}
	return u.value.([]SymbolInformation), true
}

// Array1 returns the DocumentSymbol[] variant value and true if selected.
func (u OrResultTextDocumentDocumentSymbol) Array1() ([]DocumentSymbol, bool) {
	if u.tag != 1 {
		var zero []DocumentSymbol
		return zero, false
	}
	return u.value.([]DocumentSymbol), true
}

func (u OrResultTextDocumentFoldingRange) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentFoldingRange) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || foldingRangeMatches(raw.([]any)[0])) {
		var v []FoldingRange
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentFoldingRangeArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the FoldingRange[] variant value and true if selected.
func (u OrResultTextDocumentFoldingRange) Array0() ([]FoldingRange, bool) {
	if u.tag != 0 {
		var zero []FoldingRange
		return zero, false
	}
	return u.value.([]FoldingRange), true
}

func (u OrResultTextDocumentFormatting) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentFormatting) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || textEditMatches(raw.([]any)[0])) {
		var v []TextEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentFormattingArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the TextEdit[] variant value and true if selected.
func (u OrResultTextDocumentFormatting) Array0() ([]TextEdit, bool) {
	if u.tag != 0 {
		var zero []TextEdit
		return zero, false
	}
	return u.value.([]TextEdit), true
}

func (u OrResultTextDocumentHover) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentHover) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if hoverMatches(raw) {
		var v Hover
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentHoverHover(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Hover returns the Hover variant value and true if selected.
func (u OrResultTextDocumentHover) Hover() (Hover, bool) {
	if u.tag != 0 {
		var zero Hover
		return zero, false
	}
	return u.value.(Hover), true
}

func (u OrResultTextDocumentImplementation) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentImplementation) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if definitionMatches(raw) {
		var v Definition
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentImplementationDefinition(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || definitionLinkMatches(raw.([]any)[0])) {
		var v []DefinitionLink
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentImplementationArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Definition returns the Definition variant value and true if selected.
func (u OrResultTextDocumentImplementation) Definition() (Definition, bool) {
	if u.tag != 0 {
		var zero Definition
		return zero, false
	}
	return u.value.(Definition), true
}

// Array1 returns the DefinitionLink[] variant value and true if selected.
func (u OrResultTextDocumentImplementation) Array1() ([]DefinitionLink, bool) {
	if u.tag != 1 {
		var zero []DefinitionLink
		return zero, false
	}
	return u.value.([]DefinitionLink), true
}

func (u OrResultTextDocumentInlayHint) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentInlayHint) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || inlayHintMatches(raw.([]any)[0])) {
		var v []InlayHint
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentInlayHintArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the InlayHint[] variant value and true if selected.
func (u OrResultTextDocumentInlayHint) Array0() ([]InlayHint, bool) {
	if u.tag != 0 {
		var zero []InlayHint
		return zero, false
	}
	return u.value.([]InlayHint), true
}

func (u OrResultTextDocumentInlineCompletion) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentInlineCompletion) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if inlineCompletionListMatches(raw) {
		var v InlineCompletionList
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentInlineCompletionInlineCompletionList(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || inlineCompletionItemMatches(raw.([]any)[0])) {
		var v []InlineCompletionItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentInlineCompletionArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// InlineCompletionList returns the InlineCompletionList variant value and true if selected.
func (u OrResultTextDocumentInlineCompletion) InlineCompletionList() (InlineCompletionList, bool) {
	if u.tag != 0 {
		var zero InlineCompletionList
		return zero, false
	}
	return u.value.(InlineCompletionList), true
}

// Array1 returns the InlineCompletionItem[] variant value and true if selected.
func (u OrResultTextDocumentInlineCompletion) Array1() ([]InlineCompletionItem, bool) {
	if u.tag != 1 {
		var zero []InlineCompletionItem
		return zero, false
	}
	return u.value.([]InlineCompletionItem), true
}

func (u OrResultTextDocumentInlineValue) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentInlineValue) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || inlineValueMatches(raw.([]any)[0])) {
		var v []InlineValue
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentInlineValueArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the InlineValue[] variant value and true if selected.
func (u OrResultTextDocumentInlineValue) Array0() ([]InlineValue, bool) {
	if u.tag != 0 {
		var zero []InlineValue
		return zero, false
	}
	return u.value.([]InlineValue), true
}

func (u OrResultTextDocumentLinkedEditingRange) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentLinkedEditingRange) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if linkedEditingRangesMatches(raw) {
		var v LinkedEditingRanges
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentLinkedEditingRangeLinkedEditingRanges(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// LinkedEditingRanges returns the LinkedEditingRanges variant value and true if selected.
func (u OrResultTextDocumentLinkedEditingRange) LinkedEditingRanges() (LinkedEditingRanges, bool) {
	if u.tag != 0 {
		var zero LinkedEditingRanges
		return zero, false
	}
	return u.value.(LinkedEditingRanges), true
}

func (u OrResultTextDocumentMoniker) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentMoniker) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || monikerMatches(raw.([]any)[0])) {
		var v []Moniker
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentMonikerArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the Moniker[] variant value and true if selected.
func (u OrResultTextDocumentMoniker) Array0() ([]Moniker, bool) {
	if u.tag != 0 {
		var zero []Moniker
		return zero, false
	}
	return u.value.([]Moniker), true
}

func (u OrResultTextDocumentOnTypeFormatting) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentOnTypeFormatting) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || textEditMatches(raw.([]any)[0])) {
		var v []TextEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentOnTypeFormattingArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the TextEdit[] variant value and true if selected.
func (u OrResultTextDocumentOnTypeFormatting) Array0() ([]TextEdit, bool) {
	if u.tag != 0 {
		var zero []TextEdit
		return zero, false
	}
	return u.value.([]TextEdit), true
}

func (u OrResultTextDocumentPrepareCallHierarchy) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentPrepareCallHierarchy) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || callHierarchyItemMatches(raw.([]any)[0])) {
		var v []CallHierarchyItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentPrepareCallHierarchyArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the CallHierarchyItem[] variant value and true if selected.
func (u OrResultTextDocumentPrepareCallHierarchy) Array0() ([]CallHierarchyItem, bool) {
	if u.tag != 0 {
		var zero []CallHierarchyItem
		return zero, false
	}
	return u.value.([]CallHierarchyItem), true
}

func (u OrResultTextDocumentPrepareRename) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentPrepareRename) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if prepareRenameResultMatches(raw) {
		var v PrepareRenameResult
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentPrepareRenamePrepareRenameResult(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// PrepareRenameResult returns the PrepareRenameResult variant value and true if selected.
func (u OrResultTextDocumentPrepareRename) PrepareRenameResult() (PrepareRenameResult, bool) {
	if u.tag != 0 {
		var zero PrepareRenameResult
		return zero, false
	}
	return u.value.(PrepareRenameResult), true
}

func (u OrResultTextDocumentPrepareTypeHierarchy) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentPrepareTypeHierarchy) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || typeHierarchyItemMatches(raw.([]any)[0])) {
		var v []TypeHierarchyItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentPrepareTypeHierarchyArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the TypeHierarchyItem[] variant value and true if selected.
func (u OrResultTextDocumentPrepareTypeHierarchy) Array0() ([]TypeHierarchyItem, bool) {
	if u.tag != 0 {
		var zero []TypeHierarchyItem
		return zero, false
	}
	return u.value.([]TypeHierarchyItem), true
}

func (u OrResultTextDocumentRangeFormatting) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentRangeFormatting) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || textEditMatches(raw.([]any)[0])) {
		var v []TextEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentRangeFormattingArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the TextEdit[] variant value and true if selected.
func (u OrResultTextDocumentRangeFormatting) Array0() ([]TextEdit, bool) {
	if u.tag != 0 {
		var zero []TextEdit
		return zero, false
	}
	return u.value.([]TextEdit), true
}

func (u OrResultTextDocumentRangesFormatting) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentRangesFormatting) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || textEditMatches(raw.([]any)[0])) {
		var v []TextEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentRangesFormattingArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the TextEdit[] variant value and true if selected.
func (u OrResultTextDocumentRangesFormatting) Array0() ([]TextEdit, bool) {
	if u.tag != 0 {
		var zero []TextEdit
		return zero, false
	}
	return u.value.([]TextEdit), true
}

func (u OrResultTextDocumentReferences) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentReferences) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || locationMatches(raw.([]any)[0])) {
		var v []Location
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentReferencesArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the Location[] variant value and true if selected.
func (u OrResultTextDocumentReferences) Array0() ([]Location, bool) {
	if u.tag != 0 {
		var zero []Location
		return zero, false
	}
	return u.value.([]Location), true
}

func (u OrResultTextDocumentRename) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentRename) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if workspaceEditMatches(raw) {
		var v WorkspaceEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentRenameWorkspaceEdit(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// WorkspaceEdit returns the WorkspaceEdit variant value and true if selected.
func (u OrResultTextDocumentRename) WorkspaceEdit() (WorkspaceEdit, bool) {
	if u.tag != 0 {
		var zero WorkspaceEdit
		return zero, false
	}
	return u.value.(WorkspaceEdit), true
}

func (u OrResultTextDocumentSelectionRange) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentSelectionRange) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || selectionRangeMatches(raw.([]any)[0])) {
		var v []SelectionRange
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentSelectionRangeArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the SelectionRange[] variant value and true if selected.
func (u OrResultTextDocumentSelectionRange) Array0() ([]SelectionRange, bool) {
	if u.tag != 0 {
		var zero []SelectionRange
		return zero, false
	}
	return u.value.([]SelectionRange), true
}

func (u OrResultTextDocumentSemanticTokensFull) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentSemanticTokensFull) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if semanticTokensMatches(raw) {
		var v SemanticTokens
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentSemanticTokensFullSemanticTokens(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// SemanticTokens returns the SemanticTokens variant value and true if selected.
func (u OrResultTextDocumentSemanticTokensFull) SemanticTokens() (SemanticTokens, bool) {
	if u.tag != 0 {
		var zero SemanticTokens
		return zero, false
	}
	return u.value.(SemanticTokens), true
}

func (u OrResultTextDocumentSemanticTokensFullDelta) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentSemanticTokensFullDelta) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if semanticTokensMatches(raw) {
		var v SemanticTokens
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentSemanticTokensFullDeltaSemanticTokens(v)
		return nil
	}
	if semanticTokensDeltaMatches(raw) {
		var v SemanticTokensDelta
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentSemanticTokensFullDeltaSemanticTokensDelta(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// SemanticTokens returns the SemanticTokens variant value and true if selected.
func (u OrResultTextDocumentSemanticTokensFullDelta) SemanticTokens() (SemanticTokens, bool) {
	if u.tag != 0 {
		var zero SemanticTokens
		return zero, false
	}
	return u.value.(SemanticTokens), true
}

// SemanticTokensDelta returns the SemanticTokensDelta variant value and true if selected.
func (u OrResultTextDocumentSemanticTokensFullDelta) SemanticTokensDelta() (SemanticTokensDelta, bool) {
	if u.tag != 1 {
		var zero SemanticTokensDelta
		return zero, false
	}
	return u.value.(SemanticTokensDelta), true
}

func (u OrResultTextDocumentSemanticTokensRange) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentSemanticTokensRange) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if semanticTokensMatches(raw) {
		var v SemanticTokens
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentSemanticTokensRangeSemanticTokens(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// SemanticTokens returns the SemanticTokens variant value and true if selected.
func (u OrResultTextDocumentSemanticTokensRange) SemanticTokens() (SemanticTokens, bool) {
	if u.tag != 0 {
		var zero SemanticTokens
		return zero, false
	}
	return u.value.(SemanticTokens), true
}

func (u OrResultTextDocumentSignatureHelp) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentSignatureHelp) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if signatureHelpMatches(raw) {
		var v SignatureHelp
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentSignatureHelpSignatureHelp(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// SignatureHelp returns the SignatureHelp variant value and true if selected.
func (u OrResultTextDocumentSignatureHelp) SignatureHelp() (SignatureHelp, bool) {
	if u.tag != 0 {
		var zero SignatureHelp
		return zero, false
	}
	return u.value.(SignatureHelp), true
}

func (u OrResultTextDocumentTypeDefinition) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentTypeDefinition) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if definitionMatches(raw) {
		var v Definition
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentTypeDefinitionDefinition(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || definitionLinkMatches(raw.([]any)[0])) {
		var v []DefinitionLink
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentTypeDefinitionArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Definition returns the Definition variant value and true if selected.
func (u OrResultTextDocumentTypeDefinition) Definition() (Definition, bool) {
	if u.tag != 0 {
		var zero Definition
		return zero, false
	}
	return u.value.(Definition), true
}

// Array1 returns the DefinitionLink[] variant value and true if selected.
func (u OrResultTextDocumentTypeDefinition) Array1() ([]DefinitionLink, bool) {
	if u.tag != 1 {
		var zero []DefinitionLink
		return zero, false
	}
	return u.value.([]DefinitionLink), true
}

func (u OrResultTextDocumentWillSaveWaitUntil) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTextDocumentWillSaveWaitUntil) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || textEditMatches(raw.([]any)[0])) {
		var v []TextEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTextDocumentWillSaveWaitUntilArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the TextEdit[] variant value and true if selected.
func (u OrResultTextDocumentWillSaveWaitUntil) Array0() ([]TextEdit, bool) {
	if u.tag != 0 {
		var zero []TextEdit
		return zero, false
	}
	return u.value.([]TextEdit), true
}

func (u OrResultTypeHierarchySubtypes) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTypeHierarchySubtypes) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || typeHierarchyItemMatches(raw.([]any)[0])) {
		var v []TypeHierarchyItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTypeHierarchySubtypesArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the TypeHierarchyItem[] variant value and true if selected.
func (u OrResultTypeHierarchySubtypes) Array0() ([]TypeHierarchyItem, bool) {
	if u.tag != 0 {
		var zero []TypeHierarchyItem
		return zero, false
	}
	return u.value.([]TypeHierarchyItem), true
}

func (u OrResultTypeHierarchySupertypes) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultTypeHierarchySupertypes) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || typeHierarchyItemMatches(raw.([]any)[0])) {
		var v []TypeHierarchyItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultTypeHierarchySupertypesArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the TypeHierarchyItem[] variant value and true if selected.
func (u OrResultTypeHierarchySupertypes) Array0() ([]TypeHierarchyItem, bool) {
	if u.tag != 0 {
		var zero []TypeHierarchyItem
		return zero, false
	}
	return u.value.([]TypeHierarchyItem), true
}

func (u OrResultWindowShowMessageRequest) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultWindowShowMessageRequest) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if messageActionItemMatches(raw) {
		var v MessageActionItem
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultWindowShowMessageRequestMessageActionItem(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// MessageActionItem returns the MessageActionItem variant value and true if selected.
func (u OrResultWindowShowMessageRequest) MessageActionItem() (MessageActionItem, bool) {
	if u.tag != 0 {
		var zero MessageActionItem
		return zero, false
	}
	return u.value.(MessageActionItem), true
}

func (u OrResultWorkspaceExecuteCommand) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultWorkspaceExecuteCommand) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if lSPAnyMatches(raw) {
		var v LSPAny
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultWorkspaceExecuteCommandLSPAny(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// LSPAny returns the LSPAny variant value and true if selected.
func (u OrResultWorkspaceExecuteCommand) LSPAny() (LSPAny, bool) {
	if u.tag != 0 {
		var zero LSPAny
		return zero, false
	}
	return u.value.(LSPAny), true
}

func (u OrResultWorkspaceSymbol) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	case 1:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultWorkspaceSymbol) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || symbolInformationMatches(raw.([]any)[0])) {
		var v []SymbolInformation
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultWorkspaceSymbolArray0(v)
		return nil
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || workspaceSymbolMatches(raw.([]any)[0])) {
		var v []WorkspaceSymbol
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultWorkspaceSymbolArray1(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the SymbolInformation[] variant value and true if selected.
func (u OrResultWorkspaceSymbol) Array0() ([]SymbolInformation, bool) {
	if u.tag != 0 {
		var zero []SymbolInformation
		return zero, false
	}
	return u.value.([]SymbolInformation), true
}

// Array1 returns the WorkspaceSymbol[] variant value and true if selected.
func (u OrResultWorkspaceSymbol) Array1() ([]WorkspaceSymbol, bool) {
	if u.tag != 1 {
		var zero []WorkspaceSymbol
		return zero, false
	}
	return u.value.([]WorkspaceSymbol), true
}

func (u OrResultWorkspaceWillCreateFiles) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultWorkspaceWillCreateFiles) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if workspaceEditMatches(raw) {
		var v WorkspaceEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultWorkspaceWillCreateFilesWorkspaceEdit(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// WorkspaceEdit returns the WorkspaceEdit variant value and true if selected.
func (u OrResultWorkspaceWillCreateFiles) WorkspaceEdit() (WorkspaceEdit, bool) {
	if u.tag != 0 {
		var zero WorkspaceEdit
		return zero, false
	}
	return u.value.(WorkspaceEdit), true
}

func (u OrResultWorkspaceWillDeleteFiles) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultWorkspaceWillDeleteFiles) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if workspaceEditMatches(raw) {
		var v WorkspaceEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultWorkspaceWillDeleteFilesWorkspaceEdit(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// WorkspaceEdit returns the WorkspaceEdit variant value and true if selected.
func (u OrResultWorkspaceWillDeleteFiles) WorkspaceEdit() (WorkspaceEdit, bool) {
	if u.tag != 0 {
		var zero WorkspaceEdit
		return zero, false
	}
	return u.value.(WorkspaceEdit), true
}

func (u OrResultWorkspaceWillRenameFiles) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultWorkspaceWillRenameFiles) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if workspaceEditMatches(raw) {
		var v WorkspaceEdit
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultWorkspaceWillRenameFilesWorkspaceEdit(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// WorkspaceEdit returns the WorkspaceEdit variant value and true if selected.
func (u OrResultWorkspaceWillRenameFiles) WorkspaceEdit() (WorkspaceEdit, bool) {
	if u.tag != 0 {
		var zero WorkspaceEdit
		return zero, false
	}
	return u.value.(WorkspaceEdit), true
}

func (u OrResultWorkspaceWorkspaceFolders) MarshalJSON() ([]byte, error) {
	switch u.tag {
	case 0:
		return json.Marshal(u.value)
	default:
		return nil, fmt.Errorf("union has no selected variant")
	}
}

func (u *OrResultWorkspaceWorkspaceFolders) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("union cannot unmarshal empty data")
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if isArray(raw) && (len(raw.([]any)) == 0 || workspaceFolderMatches(raw.([]any)[0])) {
		var v []WorkspaceFolder
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*u = NewOrResultWorkspaceWorkspaceFoldersArray0(v)
		return nil
	}
	return fmt.Errorf("data does not match any variant of %s", string(data))
}

// Array0 returns the WorkspaceFolder[] variant value and true if selected.
func (u OrResultWorkspaceWorkspaceFolders) Array0() ([]WorkspaceFolder, bool) {
	if u.tag != 0 {
		var zero []WorkspaceFolder
		return zero, false
	}
	return u.value.([]WorkspaceFolder), true
}
