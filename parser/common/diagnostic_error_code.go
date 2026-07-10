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
package common

import "ballerina/tools/diagnostics"

type DiagnosticErrorCode struct {
	diagnosticId string
	messageKey   string
}

// Constants ported from io.ballerina.compiler.internal.diagnostics.DiagnosticErrorCode
// Generic syntax error
var ERROR_SYNTAX_ERROR = DiagnosticErrorCode{diagnosticId: "BCE0000", messageKey: "error.syntax.error"}

// Missing tokens
var ERROR_MISSING_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0001", messageKey: "error.missing.token"}
var ERROR_MISSING_SEMICOLON_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0002", messageKey: "error.missing.semicolon.token"}
var ERROR_MISSING_COLON_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0003", messageKey: "error.missing.colon.token"}
var ERROR_MISSING_OPEN_PAREN_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0004", messageKey: "error.missing.open.paren.token"}
var ERROR_MISSING_CLOSE_PAREN_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0005", messageKey: "error.missing.close.paren.token"}
var ERROR_MISSING_OPEN_BRACE_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0006", messageKey: "error.missing.open.brace.token"}
var ERROR_MISSING_CLOSE_BRACE_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0007", messageKey: "error.missing.close.brace.token"}
var ERROR_MISSING_OPEN_BRACKET_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0008", messageKey: "error.missing.open.bracket.token"}
var ERROR_MISSING_CLOSE_BRACKET_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0009", messageKey: "error.missing.close.bracket.token"}
var ERROR_MISSING_EQUAL_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0010", messageKey: "error.missing.equal.token"}
var ERROR_MISSING_COMMA_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0011", messageKey: "error.missing.comma.token"}
var ERROR_MISSING_BINARY_OPERATOR = DiagnosticErrorCode{diagnosticId: "BCE0012", messageKey: "error.missing.binary.operator"}
var ERROR_MISSING_SLASH_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0013", messageKey: "error.missing.slash.token"}
var ERROR_MISSING_AT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0014", messageKey: "error.missing.at.token"}
var ERROR_MISSING_QUESTION_MARK_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0015", messageKey: "error.missing.question.mark.token"}
var ERROR_MISSING_GT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0016", messageKey: "error.missing.gt.token"}
var ERROR_MISSING_GT_EQUAL_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0017", messageKey: "error.missing.gt.equal.token"}
var ERROR_MISSING_LT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0018", messageKey: "error.missing.lt.token"}
var ERROR_MISSING_LT_EQUAL_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0019", messageKey: "error.missing.lt.equal.token"}
var ERROR_MISSING_RIGHT_DOUBLE_ARROW_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0020", messageKey: "error.missing.right.double.arrow.token"}
var ERROR_MISSING_XML_COMMENT_END_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0021", messageKey: "error.missing.xml.comment.end.token"}
var ERROR_MISSING_XML_PI_END_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0022", messageKey: "error.missing.xml.pi.end.token"}
var ERROR_MISSING_DOUBLE_QUOTE_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0023", messageKey: "error.missing.double.quote.token"}
var ERROR_MISSING_BACKTICK_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0024", messageKey: "error.missing.backtick.token"}
var ERROR_MISSING_OPEN_BRACE_PIPE_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0025", messageKey: "error.missing.open.brace.pipe.token"}
var ERROR_MISSING_CLOSE_BRACE_PIPE_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0026", messageKey: "error.missing.close.brace.pipe.token"}
var ERROR_MISSING_ASTERISK_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0027", messageKey: "error.missing.asterisk.token"}
var ERROR_MISSING_PIPE_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0028", messageKey: "error.missing.pipe.token"}
var ERROR_MISSING_DOT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0029", messageKey: "error.missing.dot.token"}
var ERROR_MISSING_ELLIPSIS_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0030", messageKey: "error.missing.ellipsis.token"}
var ERROR_MISSING_HASH_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0031", messageKey: "error.missing.hash.token"}
var ERROR_MISSING_SINGLE_QUOTE_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0032", messageKey: "error.missing.single.quote.token"}
var ERROR_MISSING_DOUBLE_EQUAL_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0033", messageKey: "error.missing.double.equal.token"}
var ERROR_MISSING_TRIPPLE_EQUAL_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0034", messageKey: "error.missing.tripple.equal.token"}
var ERROR_MISSING_MINUS_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0035", messageKey: "error.missing.minus.token"}
var ERROR_MISSING_PERCENT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0036", messageKey: "error.missing.percent.token"}
var ERROR_MISSING_EXCLAMATION_MARK_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0037", messageKey: "error.missing.exclamation.mark.token"}
var ERROR_MISSING_NOT_EQUAL_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0038", messageKey: "error.missing.not.equal.token"}
var ERROR_MISSING_NOT_DOUBLE_EQUAL_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0039", messageKey: "error.missing.not.double.equal.token"}
var ERROR_MISSING_BITWISE_AND_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0040", messageKey: "error.missing.bitwise.and.token"}
var ERROR_MISSING_BITWISE_XOR_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0041", messageKey: "error.missing.bitwise.xor.token"}
var ERROR_MISSING_LOGICAL_AND_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0042", messageKey: "error.missing.logical.and.token"}
var ERROR_MISSING_LOGICAL_OR_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0043", messageKey: "error.missing.logical.or.token"}
var ERROR_MISSING_NEGATION_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0044", messageKey: "error.missing.negation.token"}
var ERROR_MISSING_RIGHT_ARROW_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0045", messageKey: "error.missing.right.arrow.token"}
var ERROR_MISSING_INTERPOLATION_START_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0046", messageKey: "error.missing.interpolation.start.token"}
var ERROR_MISSING_XML_PI_START_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0047", messageKey: "error.missing.xml.pi.start.token"}
var ERROR_MISSING_XML_COMMENT_START_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0048", messageKey: "error.missing.xml.comment.start.token"}
var ERROR_MISSING_SYNC_SEND_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0049", messageKey: "error.missing.sync.send.token"}
var ERROR_MISSING_LEFT_ARROW_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0050", messageKey: "error.missing.left.arrow.token"}
var ERROR_MISSING_DOUBLE_DOT_LT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0051", messageKey: "error.missing.double.dot.lt.token"}
var ERROR_MISSING_DOUBLE_LT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0052", messageKey: "error.missing.double.lt.token"}
var ERROR_MISSING_ANNOT_CHAINING_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0053", messageKey: "error.missing.annot.chaining.token"}
var ERROR_MISSING_OPTIONAL_CHAINING_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0054", messageKey: "error.missing.optional.chaining.token"}
var ERROR_MISSING_ELVIS_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0055", messageKey: "error.missing.elvis.token"}
var ERROR_MISSING_DOT_LT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0056", messageKey: "error.missing.dot.lt.token"}
var ERROR_MISSING_SLASH_LT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0057", messageKey: "error.missing.slash.lt.token"}
var ERROR_MISSING_DOUBLE_SLASH_DOUBLE_ASTERISK_LT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0058", messageKey: "error.missing.double.slash.double.asterisk.lt.token"}
var ERROR_MISSING_SLASH_ASTERISK_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0059", messageKey: "error.missing.slash.asterisk.token"}
var ERROR_MISSING_DOUBLE_GT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0060", messageKey: "error.missing.double.gt.token"}
var ERROR_MISSING_TRIPPLE_GT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0061", messageKey: "error.missing.tripple.gt.token"}
var ERROR_MISSING_XML_CDATA_END_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0062", messageKey: "error.missing.xml.cdata.end.token"}

// Missing keywords
var ERROR_MISSING_PUBLIC_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0200", messageKey: "error.missing.public.keyword"}
var ERROR_MISSING_PRIVATE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0201", messageKey: "error.missing.private.keyword"}
var ERROR_MISSING_REMOTE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0202", messageKey: "error.missing.remote.keyword"}
var ERROR_MISSING_ABSTRACT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0203", messageKey: "error.missing.abstract.keyword"}
var ERROR_MISSING_CLIENT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0204", messageKey: "error.missing.client.keyword"}
var ERROR_MISSING_LISTENER_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0205", messageKey: "error.missing.listener.keyword"}
var ERROR_MISSING_XMLNS_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0206", messageKey: "error.missing.xmlns.keyword"}
var ERROR_MISSING_RESOURCE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0207", messageKey: "error.missing.resource.keyword"}
var ERROR_MISSING_FINAL_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0208", messageKey: "error.missing.final.keyword"}
var ERROR_MISSING_WORKER_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0209", messageKey: "error.missing.worker.keyword"}
var ERROR_MISSING_PARAMETER_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0210", messageKey: "error.missing.parameter.keyword"}
var ERROR_MISSING_RETURNS_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0211", messageKey: "error.missing.returns.keyword"}
var ERROR_MISSING_RETURN_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0212", messageKey: "error.missing.return.keyword"}
var ERROR_MISSING_TRUE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0213", messageKey: "error.missing.true.keyword"}
var ERROR_MISSING_FALSE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0214", messageKey: "error.missing.false.keyword"}
var ERROR_MISSING_ELSE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0215", messageKey: "error.missing.else.keyword"}
var ERROR_MISSING_WHILE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0216", messageKey: "error.missing.while.keyword"}
var ERROR_MISSING_CHECK_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0217", messageKey: "error.missing.check.keyword"}
var ERROR_MISSING_CHECKPANIC_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0218", messageKey: "error.missing.checkpanic.keyword"}
var ERROR_MISSING_PANIC_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0219", messageKey: "error.missing.panic.keyword"}
var ERROR_MISSING_CONTINUE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0220", messageKey: "error.missing.continue.keyword"}
var ERROR_MISSING_BREAK_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0221", messageKey: "error.missing.break.keyword"}
var ERROR_MISSING_TYPEOF_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0222", messageKey: "error.missing.typeof.keyword"}
var ERROR_MISSING_IS_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0223", messageKey: "error.missing.is.keyword"}
var ERROR_MISSING_NULL_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0224", messageKey: "error.missing.null.keyword"}
var ERROR_MISSING_LOCK_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0225", messageKey: "error.missing.lock.keyword"}
var ERROR_MISSING_FORK_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0226", messageKey: "error.missing.fork.keyword"}
var ERROR_MISSING_TRAP_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0227", messageKey: "error.missing.trap.keyword"}
var ERROR_MISSING_FOREACH_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0228", messageKey: "error.missing.foreach.keyword"}
var ERROR_MISSING_NEW_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0229", messageKey: "error.missing.new.keyword"}
var ERROR_MISSING_WHERE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0230", messageKey: "error.missing.where.keyword"}
var ERROR_MISSING_SELECT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0231", messageKey: "error.missing.select.keyword"}
var ERROR_MISSING_START_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0232", messageKey: "error.missing.start.keyword"}
var ERROR_MISSING_FLUSH_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0233", messageKey: "error.missing.flush.keyword"}
var ERROR_MISSING_WAIT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0234", messageKey: "error.missing.wait.keyword"}
var ERROR_MISSING_DO_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0235", messageKey: "error.missing.do.keyword"}
var ERROR_MISSING_TRANSACTION_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0236", messageKey: "error.missing.transaction.keyword"}
var ERROR_MISSING_TRANSACTIONAL_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0237", messageKey: "error.missing.transactional.keyword"}
var ERROR_MISSING_COMMIT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0238", messageKey: "error.missing.commit.keyword"}
var ERROR_MISSING_ROLLBACK_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0239", messageKey: "error.missing.rollback.keyword"}
var ERROR_MISSING_RETRY_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0240", messageKey: "error.missing.retry.keyword"}
var ERROR_MISSING_BASE16_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0241", messageKey: "error.missing.base16.keyword"}
var ERROR_MISSING_BASE64_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0242", messageKey: "error.missing.base64.keyword"}
var ERROR_MISSING_MATCH_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0243", messageKey: "error.missing.match.keyword"}
var ERROR_MISSING_DEFAULT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0244", messageKey: "error.missing.default.keyword"}
var ERROR_MISSING_TYPE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0245", messageKey: "error.missing.type.keyword"}
var ERROR_MISSING_ON_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0246", messageKey: "error.missing.on.keyword"}
var ERROR_MISSING_ANNOTATION_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0247", messageKey: "error.missing.annotation.keyword"}
var ERROR_MISSING_FUNCTION_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0248", messageKey: "error.missing.function.keyword"}
var ERROR_MISSING_SOURCE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0249", messageKey: "error.missing.source.keyword"}
var ERROR_MISSING_ENUM_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0250", messageKey: "error.missing.enum.keyword"}
var ERROR_MISSING_FIELD_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0251", messageKey: "error.missing.field.keyword"}
var ERROR_MISSING_VERSION_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0252", messageKey: "error.missing.version.keyword"}
var ERROR_MISSING_OBJECT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0253", messageKey: "error.missing.object.keyword"}
var ERROR_MISSING_RECORD_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0254", messageKey: "error.missing.record.keyword"}
var ERROR_MISSING_SERVICE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0255", messageKey: "error.missing.service.keyword"}
var ERROR_MISSING_AS_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0256", messageKey: "error.missing.as.keyword"}
var ERROR_MISSING_LET_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0257", messageKey: "error.missing.let.keyword"}
var ERROR_MISSING_TABLE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0258", messageKey: "error.missing.table.keyword"}
var ERROR_MISSING_KEY_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0259", messageKey: "error.missing.key.keyword"}
var ERROR_MISSING_FROM_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0260", messageKey: "error.missing.from.keyword"}
var ERROR_MISSING_IN_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0261", messageKey: "error.missing.in.keyword"}
var ERROR_MISSING_IF_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0262", messageKey: "error.missing.if.keyword"}
var ERROR_MISSING_IMPORT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0263", messageKey: "error.missing.import.keyword"}
var ERROR_MISSING_CONST_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0264", messageKey: "error.missing.const.keyword"}
var ERROR_MISSING_EXTERNAL_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0265", messageKey: "error.missing.external.keyword"}
var ERROR_MISSING_ORDER_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0266", messageKey: "error.missing.order.keyword"}
var ERROR_MISSING_BY_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0267", messageKey: "error.missing.by.keyword"}
var ERROR_MISSING_CONFLICT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0268", messageKey: "error.missing.conflict.keyword"}
var ERROR_MISSING_LIMIT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0269", messageKey: "error.missing.limit.keyword"}
var ERROR_MISSING_ASCENDING_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0270", messageKey: "error.missing.ascending.keyword"}
var ERROR_MISSING_DESCENDING_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0271", messageKey: "error.missing.descending.keyword"}
var ERROR_MISSING_JOIN_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0272", messageKey: "error.missing.join.keyword"}
var ERROR_MISSING_OUTER_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0273", messageKey: "error.missing.outer.keyword"}
var ERROR_MISSING_CLASS_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0274", messageKey: "error.missing.class.keyword"}
var ERROR_MISSING_FAIL_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0275", messageKey: "error.missing.fail.keyword"}
var ERROR_MISSING_EQUALS_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0276", messageKey: "error.missing.equals.keyword"}
var ERROR_MISSING_INT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0277", messageKey: "error.missing.int.keyword"}
var ERROR_MISSING_BYTE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0278", messageKey: "error.missing.byte.keyword"}
var ERROR_MISSING_FLOAT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0279", messageKey: "error.missing.float.keyword"}
var ERROR_MISSING_DECIMAL_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0280", messageKey: "error.missing.decimal.keyword"}
var ERROR_MISSING_STRING_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0281", messageKey: "error.missing.string.keyword"}
var ERROR_MISSING_BOOLEAN_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0282", messageKey: "error.missing.boolean.keyword"}
var ERROR_MISSING_XML_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0283", messageKey: "error.missing.xml.keyword"}
var ERROR_MISSING_JSON_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0284", messageKey: "error.missing.json.keyword"}
var ERROR_MISSING_HANDLE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0285", messageKey: "error.missing.handle.keyword"}
var ERROR_MISSING_ANY_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0286", messageKey: "error.missing.any.keyword"}
var ERROR_MISSING_ANYDATA_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0287", messageKey: "error.missing.anydata.keyword"}
var ERROR_MISSING_NEVER_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0288", messageKey: "error.missing.never.keyword"}
var ERROR_MISSING_VAR_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0289", messageKey: "error.missing.var.keyword"}
var ERROR_MISSING_MAP_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0290", messageKey: "error.missing.map.keyword"}
var ERROR_MISSING_ERROR_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0291", messageKey: "error.missing.error.keyword"}
var ERROR_MISSING_STREAM_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0292", messageKey: "error.missing.stream.keyword"}
var ERROR_MISSING_READONLY_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0293", messageKey: "error.missing.readonly.keyword"}
var ERROR_MISSING_DISTINCT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0294", messageKey: "error.missing.distinct.keyword"}
var ERROR_MISSING_RE_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0295", messageKey: "error.missing.re.keyword"}
var ERROR_MISSING_GROUP_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0296", messageKey: "error.missing.group.keyword"}
var ERROR_MISSING_COLLECT_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0297", messageKey: "error.missing.collect.keyword"}
var ERROR_MISSING_NATURAL_KEYWORD = DiagnosticErrorCode{diagnosticId: "BCE0298", messageKey: "error.missing.natural.keyword"}

// Missing other tokens
var ERROR_MISSING_IDENTIFIER = DiagnosticErrorCode{diagnosticId: "BCE0400", messageKey: "error.missing.identifier"}
var ERROR_MISSING_STRING_LITERAL = DiagnosticErrorCode{diagnosticId: "BCE0401", messageKey: "error.missing.string.literal"}
var ERROR_MISSING_DECIMAL_INTEGER_LITERAL = DiagnosticErrorCode{diagnosticId: "BCE0402", messageKey: "error.missing.decimal.integer.literal"}
var ERROR_MISSING_HEX_INTEGER_LITERAL = DiagnosticErrorCode{diagnosticId: "BCE0403", messageKey: "error.missing.hex.integer.literal"}
var ERROR_MISSING_DECIMAL_FLOATING_POINT_LITERAL = DiagnosticErrorCode{diagnosticId: "BCE0404", messageKey: "error.missing.decimal.floating.point.literal"}
var ERROR_MISSING_HEX_FLOATING_POINT_LITERAL = DiagnosticErrorCode{diagnosticId: "BCE0405", messageKey: "error.missing.hex.floating.point.literal"}
var ERROR_MISSING_XML_TEXT_CONTENT = DiagnosticErrorCode{diagnosticId: "BCE0406", messageKey: "error.missing.xml.text.content"}
var ERROR_MISSING_TEMPLATE_STRING = DiagnosticErrorCode{diagnosticId: "BCE0407", messageKey: "error.missing.template.string"}
var ERROR_MISSING_BYTE_ARRAY_CONTENT = DiagnosticErrorCode{diagnosticId: "BCE0408", messageKey: "error.missing.byte.array.content"}
var ERROR_MISSING_DIGIT_AFTER_EXPONENT_INDICATOR = DiagnosticErrorCode{diagnosticId: "BCE0409", messageKey: "error.missing.digit.after.exponent.indicator"}
var ERROR_MISSING_HEX_DIGIT_AFTER_DOT = DiagnosticErrorCode{diagnosticId: "BCE0410", messageKey: "error.missing.hex.digit.after.dot"}
var ERROR_MISSING_DOUBLE_QUOTE = DiagnosticErrorCode{diagnosticId: "BCE0411", messageKey: "error.missing.double.quote"}
var ERROR_MISSING_ENTITY_REFERENCE_NAME = DiagnosticErrorCode{diagnosticId: "BCE0412", messageKey: "error.missing.entity.reference.name"}
var ERROR_MISSING_SEMICOLON_IN_XML_REFERENCE = DiagnosticErrorCode{diagnosticId: "BCE0413", messageKey: "error.missing.semicolon.in.xml.reference"}
var ERROR_MISSING_ATTACH_POINT_NAME = DiagnosticErrorCode{diagnosticId: "BCE0414", messageKey: "error.missing.attach.point.name"}
var ERROR_MISSING_HEX_NUMBER_AFTER_HEX_INDICATOR = DiagnosticErrorCode{diagnosticId: "BCE0415", messageKey: "error.missing.hex.number.after.hex.indicator"}
var ERROR_MISSING_DIGIT_AFTER_DOT = DiagnosticErrorCode{diagnosticId: "BCE0416", messageKey: "error.missing.digit.after.dot"}
var ERROR_MISSING_RE_UNICODE_PROPERTY_VALUE = DiagnosticErrorCode{diagnosticId: "BCE0417", messageKey: "error.missing.unicode.property.value"}
var ERROR_MISSING_RE_QUANTIFIER_DIGIT = DiagnosticErrorCode{diagnosticId: "BCE0418", messageKey: "error.missing.digit.in.quantifier"}
var ERROR_MISSING_BACKSLASH = DiagnosticErrorCode{diagnosticId: "BCE0420", messageKey: "error.missing.backslash"}

// Missing non-terminal nodes
var ERROR_MISSING_FUNCTION_NAME = DiagnosticErrorCode{diagnosticId: "BCE0500", messageKey: "error.missing.function.name"}
var ERROR_MISSING_TYPE_DESC = DiagnosticErrorCode{diagnosticId: "BCE0501", messageKey: "error.missing.type.desc"}
var ERROR_MISSING_EXPRESSION = DiagnosticErrorCode{diagnosticId: "BCE0502", messageKey: "error.missing.expression"}
var ERROR_MISSING_SELECT_CLAUSE = DiagnosticErrorCode{diagnosticId: "BCE0503", messageKey: "error.missing.select.clause"}
var ERROR_MISSING_RECEIVE_FIELD_IN_RECEIVE_ACTION = DiagnosticErrorCode{diagnosticId: "BCE0504", messageKey: "error.missing.receive.field.in.receive.action"}
var ERROR_MISSING_WAIT_FIELD_IN_WAIT_ACTION = DiagnosticErrorCode{diagnosticId: "BCE0505", messageKey: "error.missing.wait.field.in.wait.action"}
var ERROR_MISSING_WAIT_FUTURE_EXPRESSION = DiagnosticErrorCode{diagnosticId: "BCE0506", messageKey: "error.missing.wait.future.expression"}
var ERROR_MISSING_ENUM_MEMBER = DiagnosticErrorCode{diagnosticId: "BCE0507", messageKey: "error.missing.enum.member"}
var ERROR_MISSING_XML_ATOMIC_NAME_PATTERN = DiagnosticErrorCode{diagnosticId: "BCE0508", messageKey: "error.missing.xml.atomic.name.pattern"}
var ERROR_MISSING_TUPLE_MEMBER = DiagnosticErrorCode{diagnosticId: "BCE0509", messageKey: "error.missing.tuple.member"}
var ERROR_MISSING_ORDER_KEY = DiagnosticErrorCode{diagnosticId: "BCE0510", messageKey: "error.missing.order.key"}
var ERROR_MISSING_ANNOTATION_ATTACH_POINT = DiagnosticErrorCode{diagnosticId: "BCE0511", messageKey: "error.missing.annotation.attach.point"}
var ERROR_MISSING_LET_VARIABLE_DECLARATION = DiagnosticErrorCode{diagnosticId: "BCE0512", messageKey: "error.missing.let.variable.declaration"}
var ERROR_MISSING_NAMED_WORKER_DECLARATION_IN_FORK_STMT = DiagnosticErrorCode{diagnosticId: "BCE0513", messageKey: "error.missing.named.worker.declaration.in.fork.stmt"}
var ERROR_MISSING_KEY_EXPR_IN_MEMBER_ACCESS_EXPR = DiagnosticErrorCode{diagnosticId: "BCE0514", messageKey: "error.missing.key.expr.in.member.access.expr"}
var ERROR_MISSING_ERROR_MESSAGE_BINDING_PATTERN = DiagnosticErrorCode{diagnosticId: "BCE0515", messageKey: "error.missing.error.message.binding.pattern"}
var ERROR_CONFIGURABLE_VARIABLE_MUST_BE_INITIALIZED_OR_REQUIRED = DiagnosticErrorCode{diagnosticId: "BCE0516", messageKey: "error.configurable.variable.must.be.initialized.or.required"}
var ERROR_MISSING_RESOURCE_PATH_IN_RESOURCE_ACCESSOR_DEFINITION = DiagnosticErrorCode{diagnosticId: "BCE0517", messageKey: "error.missing.resource.path.in.resource.accessor.definition"}
var ERROR_MISSING_RESOURCE_PATH_IN_RESOURCE_ACCESSOR_DECLARATION = DiagnosticErrorCode{diagnosticId: "BCE0518", messageKey: "error.missing.resource.path.in.resource.accessor.declaration"}
var ERROR_MISSING_ERROR_MESSAGE_IN_ERROR_CONSTRUCTOR = DiagnosticErrorCode{diagnosticId: "BCE0519", messageKey: "error.missing.error.message.in.error.constructor"}
var ERROR_MISSING_ARG_WITHIN_PARENTHESIS = DiagnosticErrorCode{diagnosticId: "BCE0520", messageKey: "error.missing.arg.within.parenthesis"}
var ERROR_MISSING_VARIABLE_NAME = DiagnosticErrorCode{diagnosticId: "BCE0521", messageKey: "error.missing.variable.name"}
var ERROR_MISSING_FIELD_NAME = DiagnosticErrorCode{diagnosticId: "BCE0522", messageKey: "error.missing.field.name"}
var ERROR_MISSING_BUILTIN_TYPE = DiagnosticErrorCode{diagnosticId: "BCE0523", messageKey: "error.missing.builtin.type"}
var ERROR_ANNOTATION_NOT_ATTACHED_TO_A_CONSTRUCT = DiagnosticErrorCode{diagnosticId: "BCE0524", messageKey: "error.annotation.not.attached.to.a.construct"}
var ERROR_DOCUMENTATION_NOT_ATTACHED_TO_A_CONSTRUCT = DiagnosticErrorCode{diagnosticId: "BCE0525", messageKey: "error.documentation.not.attached.to.a.construct"}
var ERROR_MISSING_MATCH_PATTERN = DiagnosticErrorCode{diagnosticId: "BCE0526", messageKey: "error.missing.match.pattern"}
var ERROR_MISSING_TYPE_REFERENCE = DiagnosticErrorCode{diagnosticId: "BCE0527", messageKey: "error.missing.type.reference"}
var ERROR_MISSING_BACKTICK_STRING = DiagnosticErrorCode{diagnosticId: "BCE0528", messageKey: "error.missing.backtick.string"}
var ERROR_MISSING_NAMED_ARG = DiagnosticErrorCode{diagnosticId: "BCE0529", messageKey: "error.missing.named.arg"}
var ERROR_MISSING_FIELD_MATCH_PATTERN_MEMBER = DiagnosticErrorCode{diagnosticId: "BCE0530", messageKey: "error.missing.field.match.pattern.member"}
var ERROR_MISSING_OBJECT_CONSTRUCTOR_EXPRESSION = DiagnosticErrorCode{diagnosticId: "BCE0531", messageKey: "error.missing.object.constructor.expression"}
var ERROR_MISSING_GROUPING_KEY = DiagnosticErrorCode{diagnosticId: "BCE0532", messageKey: "error.missing.grouping.key"}
var ERROR_MISSING_NATURAL_PROMPT_BLOCK = DiagnosticErrorCode{diagnosticId: "BCE0533", messageKey: "error.missing.natural.prompt.block"}

// Invalid nodes
var ERROR_INVALID_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0600", messageKey: "error.invalid.token"}
var ERROR_EXPRESSION_EXPECTED_ACTION_FOUND = DiagnosticErrorCode{diagnosticId: "BCE0601", messageKey: "error.expression.expected.action.found"}
var ERROR_ONLY_TYPE_REFERENCE_ALLOWED_AS_TYPE_INCLUSIONS = DiagnosticErrorCode{diagnosticId: "BCE0602", messageKey: "error.only.type.reference.allowed.as.type.inclusions"}
var ERROR_NAMED_WORKER_NOT_ALLOWED_HERE = DiagnosticErrorCode{diagnosticId: "BCE0603", messageKey: "error.named.worker.not.allowed.here"}
var ERROR_ONLY_NAMED_WORKERS_ALLOWED_HERE = DiagnosticErrorCode{diagnosticId: "BCE0604", messageKey: "error.only.named.workers.allowed.here"}
var ERROR_IMPORT_DECLARATION_AFTER_OTHER_DECLARATIONS = DiagnosticErrorCode{diagnosticId: "BCE0605", messageKey: "error.import.declaration.after.other.declarations"}
var ERROR_ANNOTATIONS_ATTACHED_TO_EXPRESSION = DiagnosticErrorCode{diagnosticId: "BCE0606", messageKey: "error.annotations.attached.to.expression"}
var ERROR_INVALID_EXPRESSION_IN_START_ACTION = DiagnosticErrorCode{diagnosticId: "BCE0607", messageKey: "error.invalid.expression.in.start.action"}
var ERROR_DUPLICATE_QUALIFIER = DiagnosticErrorCode{diagnosticId: "BCE0608", messageKey: "error.duplicate.qualifier"}
var ERROR_QUALIFIER_NOT_ALLOWED = DiagnosticErrorCode{diagnosticId: "BCE0609", messageKey: "error.qualifier.not.allowed"}
var ERROR_TYPE_INCLUSION_IN_OBJECT_CONSTRUCTOR = DiagnosticErrorCode{diagnosticId: "BCE0610", messageKey: "error.type.inclusion.in.object.constructor"}
var ERROR_MAPPING_CONSTRUCTOR_EXPR_AS_A_WAIT_EXPR = DiagnosticErrorCode{diagnosticId: "BCE0611", messageKey: "error.mapping.constructor.expr.as.a.wait.expr"}
var ERROR_INVALID_PARAM_LIST_IN_INFER_ANONYMOUS_FUNCTION_EXPR = DiagnosticErrorCode{diagnosticId: "BCE0612", messageKey: "error.invalid.param.list.in.infer.anonymous.function.expr"}
var ERROR_MORE_RECORD_FIELDS_AFTER_REST_FIELD = DiagnosticErrorCode{diagnosticId: "BCE0613", messageKey: "error.more.record.fields.after.rest.field"}
var ERROR_INVALID_XML_NAMESPACE_URI = DiagnosticErrorCode{diagnosticId: "BCE0614", messageKey: "error.invalid.xml.namespace.uri"}
var ERROR_INTERPOLATION_IS_NOT_ALLOWED_FOR_XML_TAG_NAMES = DiagnosticErrorCode{diagnosticId: "BCE0615", messageKey: "error.interpolation.is.not.allowed.for.xml.tag.names"}
var ERROR_INTERPOLATION_IS_NOT_ALLOWED_WITHIN_ELEMENT_TAGS = DiagnosticErrorCode{diagnosticId: "BCE0616", messageKey: "error.interpolation.is.not.allowed.within.element.tags"}
var ERROR_INTERPOLATION_IS_NOT_ALLOWED_WITHIN_XML_COMMENTS = DiagnosticErrorCode{diagnosticId: "BCE0617", messageKey: "error.interpolation.is.not.allowed.within.xml.comments"}
var ERROR_INTERPOLATION_IS_NOT_ALLOWED_WITHIN_XML_PI = DiagnosticErrorCode{diagnosticId: "BCE0618", messageKey: "error.interpolation.is.not.allowed.within.xml.pi"}
var ERROR_INVALID_EXPR_IN_ASSIGNMENT_LHS = DiagnosticErrorCode{diagnosticId: "BCE0619", messageKey: "error.invalid.expr.in.assignment.lhs"}
var ERROR_INVALID_EXPR_IN_COMPOUND_ASSIGNMENT_LHS = DiagnosticErrorCode{diagnosticId: "BCE0620", messageKey: "error.invalid.expr.in.compound.assignment.lhs"}
var ERROR_INVALID_METADATA = DiagnosticErrorCode{diagnosticId: "BCE0621", messageKey: "error.invalid.metadata"}
var ERROR_INVALID_QUALIFIER = DiagnosticErrorCode{diagnosticId: "BCE0622", messageKey: "error.invalid.qualifier"}
var ERROR_ANNOTATIONS_ATTACHED_TO_STATEMENT = DiagnosticErrorCode{diagnosticId: "BCE0623", messageKey: "error.annotations.attached.to.statement"}
var ERROR_ACTION_AS_A_WAIT_EXPR = DiagnosticErrorCode{diagnosticId: "BCE0625", messageKey: "error.action.as.a.wait.expr"}
var ERROR_INVALID_USAGE_OF_VAR = DiagnosticErrorCode{diagnosticId: "BCE0626", messageKey: "error.invalid.usage.of.var"}
var ERROR_MATCH_PATTERN_AFTER_REST_MATCH_PATTERN = DiagnosticErrorCode{diagnosticId: "BCE0627", messageKey: "error.match.pattern.after.rest.match.pattern"}
var ERROR_MATCH_PATTERN_NOT_ALLOWED = DiagnosticErrorCode{diagnosticId: "BCE0628", messageKey: "error.match.pattern.not.allowed"}
var ERROR_MATCH_STATEMENT_SHOULD_HAVE_ONE_OR_MORE_MATCH_CLAUSES = DiagnosticErrorCode{diagnosticId: "BCE0629", messageKey: "error.match.statement.should.have.one.or.more.match.clauses"}
var ERROR_PARAMETER_AFTER_THE_REST_PARAMETER = DiagnosticErrorCode{diagnosticId: "BCE0630", messageKey: "error.parameter.after.the.rest.parameter"}
var ERROR_REQUIRED_PARAMETER_AFTER_THE_DEFAULTABLE_PARAMETER = DiagnosticErrorCode{diagnosticId: "BCE0631", messageKey: "error.required.parameter.after.the.defaultable.parameter"}
var ERROR_NAMED_ARG_FOLLOWED_BY_POSITIONAL_ARG = DiagnosticErrorCode{diagnosticId: "BCE0632", messageKey: "error.named.arg.followed.by.positional.arg"}
var ERROR_REST_ARG_FOLLOWED_BY_ANOTHER_ARG = DiagnosticErrorCode{diagnosticId: "BCE0633", messageKey: "error.rest.arg.followed.by.another.arg"}
var ERROR_BINDING_PATTERN_NOT_ALLOWED = DiagnosticErrorCode{diagnosticId: "BCE0634", messageKey: "error.binding.pattern.not.allowed"}
var ERROR_INVALID_BASE16_CONTENT_IN_BYTE_ARRAY_LITERAL = DiagnosticErrorCode{diagnosticId: "BCE0635", messageKey: "error.invalid.base16.content.in.byte.array.literal"}
var ERROR_INVALID_BASE64_CONTENT_IN_BYTE_ARRAY_LITERAL = DiagnosticErrorCode{diagnosticId: "BCE0636", messageKey: "error.invalid.base64.content.in.byte.array.literal"}
var ERROR_INVALID_CONTENT_IN_BYTE_ARRAY_LITERAL = DiagnosticErrorCode{diagnosticId: "BCE0637", messageKey: "error.invalid.content.in.byte.array.literal"}
var ERROR_INVALID_EXPRESSION_STATEMENT = DiagnosticErrorCode{diagnosticId: "BCE0638", messageKey: "error.invalid.expression.statement"}
var ERROR_INVALID_ARRAY_LENGTH = DiagnosticErrorCode{diagnosticId: "BCE0639", messageKey: "error.invalid.array.length"}
var ERROR_SELECT_CLAUSE_IN_QUERY_ACTION = DiagnosticErrorCode{diagnosticId: "BCE0640", messageKey: "error.select.clause.in.query.action"}
var ERROR_MORE_CLAUSES_AFTER_SELECT_CLAUSE = DiagnosticErrorCode{diagnosticId: "BCE0641", messageKey: "error.more.clauses.after.select.clause"}
var ERROR_QUERY_CONSTRUCT_TYPE_IN_QUERY_ACTION = DiagnosticErrorCode{diagnosticId: "BCE0642", messageKey: "error.query.construct.type.in.query.action"}
var ERROR_NO_WHITESPACES_ALLOWED_IN_RIGHT_SHIFT_OP = DiagnosticErrorCode{diagnosticId: "BCE0643", messageKey: "error.no.whitespaces.allowed.in.right.shift.op"}
var ERROR_NO_WHITESPACES_ALLOWED_IN_UNSIGNED_RIGHT_SHIFT_OP = DiagnosticErrorCode{diagnosticId: "BCE0644", messageKey: "error.no.whitespaces.allowed.in.unsigned.right.shift.op"}
var ERROR_INVALID_WHITESPACE_IN_SLASH_LT_TOKEN = DiagnosticErrorCode{diagnosticId: "BCE0645", messageKey: "error.invalid.whitespace.in.slash.lt.token"}
var ERROR_LOCAL_TYPE_DEFINITION_NOT_ALLOWED = DiagnosticErrorCode{diagnosticId: "BCE0646", messageKey: "error.local.type.definition.not.allowed"}
var ERROR_LEADING_ZEROS_IN_NUMERIC_LITERALS = DiagnosticErrorCode{diagnosticId: "BCE0647", messageKey: "error.leading.zeros.in.numeric.literals"}
var ERROR_INVALID_STRING_NUMERIC_ESCAPE_SEQUENCE = DiagnosticErrorCode{diagnosticId: "BCE0648", messageKey: "error.invalid.string.numeric.escape.sequence"}
var ERROR_INVALID_ESCAPE_SEQUENCE = DiagnosticErrorCode{diagnosticId: "BCE0649", messageKey: "error.invalid.escape.sequence"}
var ERROR_INVALID_WHITESPACE_BEFORE = DiagnosticErrorCode{diagnosticId: "BCE0650", messageKey: "error.invalid.whitespace.before"}
var ERROR_INVALID_WHITESPACE_AFTER = DiagnosticErrorCode{diagnosticId: "BCE0651", messageKey: "error.invalid.whitespace.after"}
var ERROR_INVALID_XML_NAME = DiagnosticErrorCode{diagnosticId: "BCE0652", messageKey: "error.invalid.xml.name"}
var ERROR_INVALID_CHARACTER_IN_XML_ATTRIBUTE_VALUE = DiagnosticErrorCode{diagnosticId: "BCE0653", messageKey: "error.invalid.character.in.xml.attribute.value"}
var ERROR_INVALID_ENTITY_REFERENCE_NAME_START = DiagnosticErrorCode{diagnosticId: "BCE0654", messageKey: "error.invalid.entity.reference.name.start"}
var ERROR_DOUBLE_HYPHEN_NOT_ALLOWED_WITHIN_XML_COMMENT = DiagnosticErrorCode{diagnosticId: "BCE0655", messageKey: "error.double.hyphen.not.allowed.within.xml.comment"}
var ERROR_MORE_THAN_ONE_OBJECT_NETWORK_QUALIFIERS = DiagnosticErrorCode{diagnosticId: "BCE0657", messageKey: "error.more.than.one.object.network.qualifiers"}
var ERROR_REMOTE_METHOD_HAS_A_VISIBILITY_QUALIFIER = DiagnosticErrorCode{diagnosticId: "BCE0658", messageKey: "error.remote.method.has.a.visibility.qualifier"}
var ERROR_PRIVATE_QUALIFIER_IN_OBJECT_MEMBER_DESCRIPTOR = DiagnosticErrorCode{diagnosticId: "BCE0659", messageKey: "error.private.qualifier.in.object.member.descriptor"}
var ERROR_RESOURCE_PATH_IN_FUNCTION_DEFINITION = DiagnosticErrorCode{diagnosticId: "BCE0660", messageKey: "error.resource.path.in.function.definition"}
var ERROR_RESOURCE_PATH_SEGMENT_NOT_ALLOWED_AFTER_REST_PARAM = DiagnosticErrorCode{diagnosticId: "BCE0661", messageKey: "error.resource.path.segment.not.allowed.after.rest.param"}
var ERROR_REST_ARG_IN_ERROR_CONSTRUCTOR = DiagnosticErrorCode{diagnosticId: "BCE0662", messageKey: "error.rest.arg.in.error.constructor"}
var ERROR_ADDITIONAL_POSITIONAL_ARG_IN_ERROR_CONSTRUCTOR = DiagnosticErrorCode{diagnosticId: "BCE0663", messageKey: "error.additional.positional.arg.in.error.constructor"}
var ERROR_DEFAULTABLE_PARAMETER_CANNOT_BE_INCLUDED_RECORD_PARAMETER = DiagnosticErrorCode{diagnosticId: "BCE0664", messageKey: "error.defaultable.parameter.cannot.be.included.record.parameter"}
var ERROR_INCOMPLETE_QUOTED_IDENTIFIER = DiagnosticErrorCode{diagnosticId: "BCE0665", messageKey: "error.incomplete.quoted.identifier"}
var ERROR_INCLUSIVE_RECORD_TYPE_CANNOT_CONTAIN_REST_FIELD = DiagnosticErrorCode{diagnosticId: "BCE0666", messageKey: "error.inclusive.record.type.cannot.contain.rest.field"}
var ERROR_VARIABLE_DECL_HAVING_BP_MUST_BE_INITIALIZED = DiagnosticErrorCode{diagnosticId: "BCE0667", messageKey: "error.variable.decl.having.bp.must.be.initialized"}
var ERROR_ISOLATED_VAR_CANNOT_BE_DECLARED_AS_PUBLIC = DiagnosticErrorCode{diagnosticId: "BCE0668", messageKey: "error.isolated.var.cannot.be.declared.as.public"}
var ERROR_VARIABLE_DECLARED_WITH_VAR_CANNOT_BE_PUBLIC = DiagnosticErrorCode{diagnosticId: "BCE0669", messageKey: "error.variable.declared.with.var.cannot.be.public"}
var ERROR_FIELD_BP_INSIDE_LIST_BP = DiagnosticErrorCode{diagnosticId: "BCE0670", messageKey: "error.field.binding.pattern.inside.list.binding.pattern"}
var ERROR_INVALID_EXPRESSION_EXPECTED_CALL_EXPRESSION = DiagnosticErrorCode{diagnosticId: "BCE0671", messageKey: "error.invalid.expression.expected.a.call.expression"}
var ERROR_TYPE_DESC_AFTER_REST_DESCRIPTOR = DiagnosticErrorCode{diagnosticId: "BCE0672", messageKey: "error.type.desc.after.rest.descriptor"}
var ERROR_CONFIGURABLE_VAR_IMPLICITLY_FINAL = DiagnosticErrorCode{diagnosticId: "BCE0673", messageKey: "error.configurable.var.implicitly.final"}
var ERROR_LOCAL_CONST_DECL_NOT_ALLOWED = DiagnosticErrorCode{diagnosticId: "BCE0674", messageKey: "error.local.const.decl.not.allowed"}
var ERROR_FIELD_INITIALIZATION_NOT_ALLOWED_IN_OBJECT_TYPE = DiagnosticErrorCode{diagnosticId: "BCE0675", messageKey: "error.field.initialization.not.allowed.in.object.type"}
var ERROR_INTERVENING_WHITESPACES_ARE_NOT_ALLOWED = DiagnosticErrorCode{diagnosticId: "BCE0676", messageKey: "error.intervening.whitespaces.are.not.allowed"}
var ERROR_INVALID_BINDING_PATTERN = DiagnosticErrorCode{diagnosticId: "BCE0677", messageKey: "error.invalid.binding.pattern"}
var ERROR_RESOURCE_PATH_CANNOT_BEGIN_WITH_SLASH = DiagnosticErrorCode{diagnosticId: "BCE0678", messageKey: "error.resource.path.cannot.begin.with.slash"}
var REST_PARAMETER_CANNOT_BE_INCLUDED_RECORD_PARAMETER = DiagnosticErrorCode{diagnosticId: "BCE0679", messageKey: "error.rest.parameter.cannot.be.included.record.parameter"}
var RESOURCE_ACCESS_SEGMENT_IS_NOT_ALLOWED_AFTER_REST_SEGMENT = DiagnosticErrorCode{diagnosticId: "BCE0680", messageKey: "error.resource.access.segment.is.not.allowed.after.rest.segment"}
var ERROR_INVALID_TOKEN_IN_REG_EXP = DiagnosticErrorCode{diagnosticId: "BCE0681", messageKey: "error.invalid.token.in.reg.exp"}
var ERROR_INVALID_FLAG_IN_REG_EXP = DiagnosticErrorCode{diagnosticId: "BCE0682", messageKey: "error.invalid.flag.in.reg.exp"}
var ERROR_INVALID_QUANTIFIER_IN_REG_EXP = DiagnosticErrorCode{diagnosticId: "BCE0683", messageKey: "error.invalid.quantifier.in.reg.exp"}
var ERROR_ANNOTATIONS_NOT_ALLOWED_FOR_TUPLE_REST_DESCRIPTOR = DiagnosticErrorCode{diagnosticId: "BCE0684", messageKey: "error.annotations.not.allowed.for.tuple.rest.descriptor"}
var ERROR_INVALID_RE_SYNTAX_CHAR = DiagnosticErrorCode{diagnosticId: "BCE0685", messageKey: "error.invalid.syntax.char"}
var ERROR_MORE_CLAUSES_AFTER_COLLECT_CLAUSE = DiagnosticErrorCode{diagnosticId: "BCE0686", messageKey: "error.more.clauses.after.collect.clause"}
var ERROR_COLLECT_CLAUSE_IN_QUERY_ACTION = DiagnosticErrorCode{diagnosticId: "BCE0687", messageKey: "error.collect.clause.in.query.action"}

func (d *DiagnosticErrorCode) DiagnosticId() string {
	return d.diagnosticId
}

func (d *DiagnosticErrorCode) MessageKey() string {
	return d.messageKey
}

func (d *DiagnosticErrorCode) Severity() diagnostics.DiagnosticSeverity {
	return diagnostics.Error
}

func (d *DiagnosticErrorCode) Equals(code diagnostics.DiagnosticCode) bool {
	return d.messageKey == code.DiagnosticId()
}
