# Supported ballerina library features

Subset 4 extends the released [subset 3](subset3.md) with the new `ballerina/data.yaml` package, which parses YAML into, and writes YAML from, values typed from the call site.

## [data.yaml](https://github.com/ballerina-platform/module-ballerina-data.yaml)

Types: `yaml:Options`, `yaml:WriteConfig`, `yaml:YAMLSchema`, `yaml:NameConfig`, the `yaml:Name` annotation, and `yaml:Error`.

| Function | Notes |
|---|---|
| `yaml:parseString(s, options?, t?)` | Parse YAML text into `t`, a subtype of `anydata` inferred from the call site; returns `t\|yaml:Error` |
| `yaml:parseBytes(s, options?, t?)` | As `parseString`, decoding the bytes as UTF-8 |
| `yaml:parseStream(s, options?, t?)` | As `parseBytes`, reading a `stream<byte[], error?>` to its end |
| `yaml:toYamlString(yamlValue, config?)` | Write a value as YAML lines joined with `\n`; returns `string\|yaml:Error` |

Parsing reads block and flow collections, every scalar style, comments, anchors and aliases, the core tags and directives, and several documents, which bind to list, tuple, `json` and `anydata` targets. Every `Options` field is supported: the schema, `allowAnchorRedefinition`, `allowMapEntryRedefinition` and `allowDataProjection` with `nilAsOptionalField`, `absentAsNilableType` and `enableYamlStreamReorder`; `enableConstraintValidation` is accepted and ignored. Every `WriteConfig` field except `blockLevel`, which has no effect, is supported; `schema` decides which strings are quoted, where jBallerina ignores it. Scalars fold and chomp as YAML 1.2 specifies, and `toYamlString` quotes strings that would not read back unchanged. Record fields with default values are reported as missing when no key provides them, declared record fields follow the key order, and `@yaml:Name` applies only to the record type a typedesc names.

Not covered in this subset: constraint validation, `@yaml:Name` on inferred typedescs and nested records, and the compiler plugin diagnostics.
