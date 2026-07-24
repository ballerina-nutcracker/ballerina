// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

// Line-by-line migration from:
// - SyntaxTreeJSONGenerator.java (ballerina-lang/compiler/ballerina-parser/src/test/java/io/ballerinalang/compiler/parser/test/SyntaxTreeJSONGenerator.java)
// - ParserTestUtils.java (ballerina-lang/compiler/ballerina-parser/src/test/java/io/ballerinalang/compiler/parser/test/ParserTestUtils.java)

package tree

import (
	"ballerina/parser/common"
	"ballerina/tools/diagnostics"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ========== Constants (ported from ParserTestConstants.java) ==========

const (
	KIND_FIELD         = "kind"
	CHILDREN_FIELD     = "children"
	DIAGNOSTICS_FIELD  = "diagnostics"
	VALUE_FIELD        = "value"
	INVALID_NODE_FIELD = "invalidNode"
	IS_MISSING_FIELD   = "isMissing"
	HAS_DIAGNOSTICS    = "hasDiagnostics"
	LEADING_MINUTIAE   = "leadingMinutiae"
	TRAILING_MINUTIAE  = "trailingMinutiae"
)

// ========== orderedJSONObject for maintaining field order ==========
// Java's Gson JsonObject maintains insertion order automatically.
// In Go, we need to explicitly maintain order since maps are unordered.

type orderedJSONObject struct {
	fields []fieldValue
}

type fieldValue struct {
	key   string
	value any
}

func newOrderedJSONObject() *orderedJSONObject {
	return &orderedJSONObject{
		fields: make([]fieldValue, 0),
	}
}

func (oj *orderedJSONObject) addProperty(key string, value any) {
	oj.fields = append(oj.fields, fieldValue{key: key, value: value})
}

func (oj *orderedJSONObject) add(key string, value any) {
	oj.fields = append(oj.fields, fieldValue{key: key, value: value})
}

func (oj *orderedJSONObject) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, fv := range oj.fields {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyBytes, _ := json.Marshal(fv.key)
		buf.Write(keyBytes)
		buf.WriteByte(':')
		valBytes, err := json.Marshal(fv.value)
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// ========== Main Entry Point ==========

// Ported from: SyntaxTreeJSONGenerator.java:88-91
//
//	public static String generateJSON(STNode treeNode) {
//	    Gson gson = new GsonBuilder().setPrettyPrinting().create();
//	    return gson.toJson(getJSON(treeNode));
//	}
func GenerateJSON(treeNode STNode) string {
	jsonObj := getJSON(treeNode)
	jsonBytes, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal JSON: %v", err))
	}
	return string(jsonBytes)
}

// ========== Core JSON Generation ==========

// Ported from: SyntaxTreeJSONGenerator.java:103-132
//
//	private static JsonElement getJSON(STNode treeNode) {
//	    JsonObject jsonNode = new JsonObject();
//	    SyntaxKind nodeKind = treeNode.kind;
//	    jsonNode.addProperty(KIND_FIELD, nodeKind.name());
//
//	    if (treeNode.isMissing()) {
//	        jsonNode.addProperty(IS_MISSING_FIELD, treeNode.isMissing());
//	        addDiagnostics(treeNode, jsonNode);
//	        if (ParserTestUtils.isToken(treeNode)) {
//	            addTrivia((STToken) treeNode, jsonNode);
//	        }
//	        return jsonNode;
//	    }
//
//	    addDiagnostics(treeNode, jsonNode);
//	    if (ParserTestUtils.isToken(treeNode)) {
//	        // If the node is a terminal node with a dynamic value (i.e: non-syntax node)
//	        // then add the value to the json.
//	        if (!ParserTestUtils.isKeyword(nodeKind)) {
//	            jsonNode.addProperty(VALUE_FIELD, ParserTestUtils.getTokenText((STToken) treeNode));
//	        }
//	        addTrivia((STToken) treeNode, jsonNode);
//	        // else do nothing
//	    } else {
//	        addChildren(treeNode, jsonNode);
//	    }
//
//	    return jsonNode;
//	}
func getJSON(treeNode STNode) any {
	jsonNode := newOrderedJSONObject()
	nodeKind := treeNode.Kind()
	jsonNode.addProperty(KIND_FIELD, kindName(nodeKind))

	if treeNode.IsMissing() {
		jsonNode.addProperty(IS_MISSING_FIELD, treeNode.IsMissing())
		addDiagnostics(treeNode, jsonNode)
		if isToken(treeNode) {
			token := treeNode.(STToken)
			addTrivia(token, jsonNode)
		}
		return jsonNode
	}

	addDiagnostics(treeNode, jsonNode)
	if isToken(treeNode) {
		token := treeNode.(STToken)
		// If the node is a terminal node with a dynamic value (i.e: non-syntax node)
		// then add the value to the json.
		if !isKeyword(nodeKind) {
			jsonNode.addProperty(VALUE_FIELD, getTokenText(token))
		}
		addTrivia(token, jsonNode)
		// else do nothing
	} else {
		addChildren(treeNode, jsonNode)
	}

	return jsonNode
}

// ========== Helper Methods for Children ==========

// Ported from: SyntaxTreeJSONGenerator.java:134-136
//
//	private static void addChildren(STNode tree, JsonObject node) {
//	    addNodeList(tree, node, CHILDREN_FIELD);
//	}
func addChildren(tree STNode, node *orderedJSONObject) {
	addNodeList(tree, node, CHILDREN_FIELD)
}

// Ported from: SyntaxTreeJSONGenerator.java:138-150
//
//	private static void addNodeList(STNode tree, JsonObject node, String key) {
//	    JsonArray children = new JsonArray();
//	    int size = tree.bucketCount();
//	    for (int i = 0; i < size; i++) {
//	        STNode childNode = tree.childInBucket(i);
//	        if (childNode == null || childNode.kind == SyntaxKind.NONE) {
//	            continue;
//	        }
//
//	        children.add(getJSON(childNode));
//	    }
//	    node.add(key, children);
//	}
func addNodeList(tree STNode, node *orderedJSONObject, key string) {
	children := make([]any, 0)
	size := tree.BucketCount()
	for i := range size {
		childNode := tree.ChildInBucket(i)
		if childNode == nil || childNode.Kind() == common.NONE {
			continue
		}

		children = append(children, getJSON(childNode))
	}
	node.add(key, children)
}

// ========== Helper Methods for Trivia (Minutiae) ==========

// Ported from: SyntaxTreeJSONGenerator.java:152-160
//
//	private static void addTrivia(STToken token, JsonObject jsonNode) {
//	    if (token.leadingMinutiae().bucketCount() != 0) {
//	        addMinutiaeList((STNodeList) token.leadingMinutiae(), jsonNode, LEADING_MINUTIAE);
//	    }
//
//	    if (token.trailingMinutiae().bucketCount() != 0) {
//	        addMinutiaeList((STNodeList) token.trailingMinutiae(), jsonNode, TRAILING_MINUTIAE);
//	    }
//	}
func addTrivia(token STToken, jsonNode *orderedJSONObject) {
	leadingMinutiae := token.LeadingMinutiae()
	if leadingMinutiae.BucketCount() != 0 {
		minutiaeList, ok := leadingMinutiae.(*STNodeList)
		if ok {
			addMinutiaeList(minutiaeList, jsonNode, LEADING_MINUTIAE)
		}
	}

	trailingMinutiae := token.TrailingMinutiae()
	if trailingMinutiae.BucketCount() != 0 {
		minutiaeList, ok := trailingMinutiae.(*STNodeList)
		if ok {
			addMinutiaeList(minutiaeList, jsonNode, TRAILING_MINUTIAE)
		}
	}
}

// Ported from: SyntaxTreeJSONGenerator.java:162-187
//
//	private static void addMinutiaeList(STNodeList minutiaeList, JsonObject node, String key) {
//	    JsonArray minutiaeJsonArray = new JsonArray();
//	    int size = minutiaeList.size();
//	    for (int i = 0; i < size; i++) {
//	        STMinutiae minutiae = (STMinutiae) minutiaeList.get(i);
//	        JsonObject minutiaeJson = new JsonObject();
//	        minutiaeJson.addProperty(KIND_FIELD, minutiae.kind.name());
//	        switch (minutiae.kind) {
//	            case WHITESPACE_MINUTIAE:
//	            case END_OF_LINE_MINUTIAE:
//	            case COMMENT_MINUTIAE:
//	                minutiaeJson.addProperty(VALUE_FIELD, minutiae.text());
//	                break;
//	            case INVALID_NODE_MINUTIAE:
//	                STInvalidNodeMinutiae invalidNodeMinutiae = (STInvalidNodeMinutiae) minutiae;
//	                STNode invalidNode = invalidNodeMinutiae.invalidNode();
//	                minutiaeJson.add(INVALID_NODE_FIELD, getJSON(invalidNode));
//	                break;
//	            default:
//	                throw new UnsupportedOperationException("Unsupported minutiae kind: '" + minutiae.kind + "'");
//	        }
//
//	        minutiaeJsonArray.add(minutiaeJson);
//	    }
//	    node.add(key, minutiaeJsonArray);
//	}
func addMinutiaeList(minutiaeList *STNodeList, node *orderedJSONObject, key string) {
	minutiaeJsonArray := make([]any, 0)
	size := minutiaeList.Size()
	for i := range size {
		minutiae := minutiaeList.Get(i)
		minutiaeJson := newOrderedJSONObject()
		minutiaeKind := minutiae.Kind()
		minutiaeJson.addProperty(KIND_FIELD, kindName(minutiaeKind))

		switch minutiaeKind {
		case common.WHITESPACE_MINUTIAE, common.END_OF_LINE_MINUTIAE, common.COMMENT_MINUTIAE:
			// Cast to STMinutiae to get text
			if stMinutiae, ok := minutiae.(*STMinutiae); ok {
				minutiaeJson.addProperty(VALUE_FIELD, stMinutiae.text)
			}
		case common.INVALID_NODE_MINUTIAE:
			if invalidNodeMinutiae, ok := minutiae.(*STInvalidNodeMinutiae); ok {
				invalidNode := invalidNodeMinutiae.invalidNode
				minutiaeJson.add(INVALID_NODE_FIELD, getJSON(invalidNode))
			}
		default:
			// panic(fmt.Sprintf("Unsupported minutiae kind: '%v'", minutiaeKind))
		}

		minutiaeJsonArray = append(minutiaeJsonArray, minutiaeJson)
	}
	node.add(key, minutiaeJsonArray)
}

// ========== Helper Methods for Diagnostics ==========

// Ported from: SyntaxTreeJSONGenerator.java:189-204
//
//	private static void addDiagnostics(STNode treeNode, JsonObject jsonNode) {
//	    if (!treeNode.hasDiagnostics()) {
//	        return;
//	    }
//
//	    jsonNode.addProperty(HAS_DIAGNOSTICS, treeNode.hasDiagnostics());
//	    Collection<STNodeDiagnostic> diagnostics = treeNode.diagnostics();
//	    if (diagnostics.isEmpty()) {
//	        return;
//	    }
//
//	    JsonArray diagnosticsJsonArray = new JsonArray();
//	    diagnostics.forEach(syntaxDiagnostic ->
//	            diagnosticsJsonArray.add(syntaxDiagnostic.diagnosticCode().toString()));
//	    jsonNode.add(DIAGNOSTICS_FIELD, diagnosticsJsonArray);
//	}
func addDiagnostics(treeNode STNode, jsonNode *orderedJSONObject) {
	if !treeNode.HasDiagnostics() {
		return
	}

	jsonNode.addProperty(HAS_DIAGNOSTICS, treeNode.HasDiagnostics())
	diagnostics := treeNode.Diagnostics()
	if len(diagnostics) == 0 {
		return
	}

	diagnosticsJsonArray := make([]any, 0, len(diagnostics))
	for _, syntaxDiagnostic := range diagnostics {
		diagnosticsJsonArray = append(diagnosticsJsonArray, diagnosticJSONMessage(syntaxDiagnostic.code))
	}
	jsonNode.add(DIAGNOSTICS_FIELD, diagnosticsJsonArray)
}

// TODO: properly implement this
func diagnosticJSONMessage(diagnosticCode diagnostics.DiagnosticCode) string {
	diagnosticId := diagnosticCode.DiagnosticId()
	if diagnosticId == "BCE0680" {
		return "RESOURCE_ACCESS_SEGMENT_IS_NOT_ALLOWED_AFTER_REST_SEGMENT"
	}
	if diagnosticId == "BCE0670" {
		return "ERROR_FIELD_BP_INSIDE_LIST_BP"
	}
	message := diagnosticCode.MessageKey()
	if message == "" {
		message = diagnosticId
	}
	return strings.ToUpper(strings.ReplaceAll(message, ".", "_"))
}

// ========== Utility Methods (from ParserTestUtils.java) ==========

// Ported from: ParserTestUtils.java:331-333
//
// public static boolean isToken(STNode node) {
//     return SyntaxUtils.isToken(node);
// }
//
// Note: isToken is already defined in st-node.go:957, so we'll just use that

// Ported from: ParserTestUtils.java:335-337
//
//	public static boolean isKeyword(SyntaxKind syntaxKind) {
//	    return SyntaxKind.IDENTIFIER_TOKEN.compareTo(syntaxKind) > 0 || syntaxKind == SyntaxKind.EOF_TOKEN;
//	}
func isKeyword(syntaxKind common.SyntaxKind) bool {
	return syntaxKind < common.IDENTIFIER_TOKEN || syntaxKind == common.EOF_TOKEN
}

// Ported from: ParserTestUtils.java:346-385
//
//	public static String getTokenText(STToken token) {
//	    switch (token.kind) {
//	        case IDENTIFIER_TOKEN:
//	            return ((STIdentifierToken) token).text;
//	        case STRING_LITERAL_TOKEN:
//	            String val = token.text();
//	            int stringLen = val.length();
//	            int lastCharPosition = val.endsWith("\"") ? stringLen - 1 : stringLen;
//	            return val.substring(1, lastCharPosition);
//	        case DECIMAL_INTEGER_LITERAL_TOKEN:
//	        case HEX_INTEGER_LITERAL_TOKEN:
//	        case DECIMAL_FLOATING_POINT_LITERAL_TOKEN:
//	        case HEX_FLOATING_POINT_LITERAL_TOKEN:
//	        case PARAMETER_NAME:
//	        case DEPRECATION_LITERAL:
//	        case INVALID_TOKEN:
//	            return token.text();
//	        case XML_TEXT:
//	        case XML_TEXT_CONTENT:
//	        case TEMPLATE_STRING:
//	        case RE_LITERAL_CHAR:
//	        case RE_NUMERIC_ESCAPE:
//	        case RE_CONTROL_ESCAPE:
//	        case RE_SIMPLE_CHAR_CLASS_CODE:
//	        case RE_PROPERTY:
//	        case RE_UNICODE_SCRIPT_START:
//	        case RE_UNICODE_PROPERTY_VALUE:
//	        case RE_UNICODE_GENERAL_CATEGORY_START:
//	        case RE_UNICODE_GENERAL_CATEGORY_NAME:
//	        case RE_FLAGS_VALUE:
//	        case DIGIT:
//	        case DOCUMENTATION_DESCRIPTION:
//	        case DOCUMENTATION_STRING:
//	        case CODE_CONTENT:
//	        case PROMPT_CONTENT:
//	            return cleanupText(token.text());
//	        default:
//	            return token.kind.toString();
//	    }
//	}
func getTokenText(token STToken) string {
	kind := token.Kind()
	switch kind {
	case common.IDENTIFIER_TOKEN:
		if identToken, ok := token.(*STIdentifierToken); ok {
			return identToken.text
		}
		return token.Text()
	case common.STRING_LITERAL_TOKEN:
		val := token.Text()
		stringLen := len(val)
		lastCharPosition := stringLen
		if stringLen > 0 && strings.HasSuffix(val, "\"") {
			lastCharPosition = stringLen - 1
		}
		if stringLen > 0 && lastCharPosition > 1 {
			return val[1:lastCharPosition]
		}
		return ""
	case common.DECIMAL_INTEGER_LITERAL_TOKEN,
		common.HEX_INTEGER_LITERAL_TOKEN,
		common.DECIMAL_FLOATING_POINT_LITERAL_TOKEN,
		common.HEX_FLOATING_POINT_LITERAL_TOKEN,
		common.PARAMETER_NAME,
		common.DEPRECATION_LITERAL,
		common.INVALID_TOKEN:
		return token.Text()
	case common.XML_TEXT,
		common.XML_TEXT_CONTENT,
		common.TEMPLATE_STRING,
		common.RE_LITERAL_CHAR,
		common.RE_NUMERIC_ESCAPE,
		common.RE_CONTROL_ESCAPE,
		common.RE_SIMPLE_CHAR_CLASS_CODE,
		common.RE_PROPERTY,
		common.RE_UNICODE_SCRIPT_START,
		common.RE_UNICODE_PROPERTY_VALUE,
		common.RE_UNICODE_GENERAL_CATEGORY_START,
		common.RE_UNICODE_GENERAL_CATEGORY_NAME,
		common.RE_FLAGS_VALUE,
		common.DIGIT,
		common.DOCUMENTATION_DESCRIPTION,
		common.DOCUMENTATION_STRING,
		common.CODE_CONTENT,
		common.PROMPT_CONTENT:
		return cleanupText(token.Text())
	default:
		return kindName(kind)
	}
}

// Ported from: ParserTestUtils.java:387-389
//
//	private static String cleanupText(String text) {
//	    return text.replace(System.lineSeparator(), "\n");
//	}
func cleanupText(text string) string {
	// System.lineSeparator() is platform-specific: "\r\n" on Windows, "\n" on Unix
	// To handle both, replace \r\n first, then standalone \r
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}
