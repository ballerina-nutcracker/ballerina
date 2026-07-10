// Copyright (c) 2026, WSO2 LLC. (http://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
//
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

// extract-surface dumps every public declaration of a Ballerina .bal source
// tree as a sorted, stable text listing of caller-observable signatures.
// It is a helper for the validate-stdlib-contract skill: run it on a
// jBallerina package's ballerina/ directory and on the matching
// lib/stdlibs/ballerina/<name>/0.0.1/go1.2/ directory, then diff the dumps.
//
// Usage: go run ./.agents/skills/validate-stdlib-contract/cmd/extract-surface <dir>
package main

import (
	"fmt"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ballerina-lang-go/context"
	"ballerina-lang-go/parser"
	"ballerina-lang-go/parser/common"
	"ballerina-lang-go/parser/tree"
	"ballerina-lang-go/semtypes"
)

var skipDirs = map[string]bool{"tests": true, "build": true, "target": true}

// items works around NodeList's pointer-receiver Iterator, which cannot be
// called on the non-addressable values accessor methods return.
func items[T tree.Node](list tree.NodeList[T]) iter.Seq[T] {
	return list.Iterator()
}

type symbol struct {
	kind string
	name string
	body []string
}

// extractor holds the current file's source so node source text can be
// recovered by TextRange slicing (ToSourceCode is not implemented on all
// facade node types).
type extractor struct {
	src string
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: extract-surface <dir-with-bal-files>")
		os.Exit(2)
	}
	files, err := collectBalFiles(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	env := context.NewCompilerEnvironment(semtypes.CreateTypeEnv(), false)
	var syms []symbol
	failed := 0
	for _, f := range files {
		fileSyms, ok := extractFile(env, f)
		if !ok {
			failed++
		}
		syms = append(syms, fileSyms...)
	}
	printSorted(syms)
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "warning: %d file(s) failed to parse — review them by hand\n", failed)
		os.Exit(1)
	}
}

func collectBalFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if p != root && (skipDirs[name] || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(p) == ".bal" {
			files = append(files, p)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func extractFile(env *context.CompilerEnvironment, path string) (syms []symbol, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: parser panic: %v\n", path, r)
			syms, ok = nil, false
		}
	}()
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s: %v\n", path, err)
		return nil, false
	}
	cx := context.NewCompilerContext(env)
	st, err := parser.GetSyntaxTree(cx, path, string(content))
	if err != nil || st == nil {
		fmt.Fprintf(os.Stderr, "warning: %s: parse failed: %v\n", path, err)
		return nil, false
	}
	mp, ok := st.RootNode.(*tree.ModulePart)
	if !ok {
		fmt.Fprintf(os.Stderr, "warning: %s: no module part\n", path)
		return nil, false
	}
	if cx.HasErrors() {
		fmt.Fprintf(os.Stderr, "warning: %s: has syntax diagnostics; extraction may be incomplete\n", path)
	}
	ex := &extractor{src: string(content)}
	for m := range items(mp.Members()) {
		if s, found := ex.member(m); found {
			syms = append(syms, s)
		}
	}
	return syms, !cx.HasErrors()
}

func (ex *extractor) text(n tree.Node) string {
	if n == nil {
		return ""
	}
	r := n.TextRange()
	if r.StartOffset < 0 || r.EndOffset > len(ex.src) || r.StartOffset > r.EndOffset {
		return ""
	}
	return norm(ex.src[r.StartOffset:r.EndOffset])
}

func (ex *extractor) member(m tree.Node) (symbol, bool) {
	switch d := m.(type) {
	case *tree.FunctionDefinition:
		quals := qualTexts(d.QualifierList())
		if !contains(quals, "public") {
			return symbol{}, false
		}
		name := d.FunctionName().Text()
		return symbol{"function", name, []string{ex.fnLine(quals, name, ex.resourcePath(d.RelativeResourcePath()), d.FunctionSignature())}}, true
	case *tree.TypeDefinitionNode:
		if !isPublicToken(d.VisibilityQualifier()) {
			return symbol{}, false
		}
		name := d.TypeName().Text()
		return symbol{"type", name, ex.typeBody(name, d.TypeDescriptor())}, true
	case *tree.ClassDefinitionNode:
		if !isPublicToken(d.VisibilityQualifier()) {
			return symbol{}, false
		}
		name := d.ClassName().Text()
		header := strings.Join(append(qualTexts(d.ClassTypeQualifiers()), "class", name), " ")
		return symbol{"class", name, append([]string{header}, ex.objectMemberLines(d.Members())...)}, true
	case *tree.ConstantDeclarationNode:
		if !isPublicToken(d.VisibilityQualifier()) {
			return symbol{}, false
		}
		name := d.VariableName().Text()
		line := "const"
		if td := d.TypeDescriptor(); td != nil {
			line += " " + ex.text(td)
		}
		line += " " + name + " = " + ex.text(d.Initializer())
		return symbol{"const", name, []string{line}}, true
	case *tree.EnumDeclarationNode:
		if !isPublicToken(d.Qualifier()) {
			return symbol{}, false
		}
		name := d.Identifier().Text()
		var members []string
		for em := range items(d.EnumMemberList()) {
			if e, ok := em.(*tree.EnumMemberNode); ok {
				line := "member " + e.Identifier().Text()
				if expr := e.ConstExprNode(); expr != nil {
					line += " = " + ex.text(expr)
				}
				members = append(members, line)
			}
		}
		sort.Strings(members)
		return symbol{"enum", name, append([]string{"enum " + name}, members...)}, true
	case *tree.AnnotationDeclarationNode:
		if !isPublicToken(d.VisibilityQualifier()) {
			return symbol{}, false
		}
		name := d.AnnotationTag().Text()
		line := "annotation"
		if d.ConstKeyword() != nil {
			line += " const"
		}
		if td := d.TypeDescriptor(); td != nil {
			line += " " + ex.text(td)
		}
		line += " " + name
		var points []string
		for ap := range items(d.AttachPoints()) {
			points = append(points, ex.text(ap))
		}
		if len(points) > 0 {
			sort.Strings(points)
			line += " on " + strings.Join(points, ", ")
		}
		return symbol{"annotation", name, []string{line}}, true
	case *tree.ListenerDeclarationNode:
		if !isPublicToken(d.VisibilityQualifier()) {
			return symbol{}, false
		}
		name := d.VariableName().Text()
		return symbol{"listener", name, []string{"listener " + ex.text(d.TypeDescriptor()) + " " + name}}, true
	case *tree.ModuleVariableDeclarationNode:
		if !isPublicToken(d.VisibilityQualifier()) {
			return symbol{}, false
		}
		binding := ex.text(d.TypedBindingPattern())
		quals := qualTexts(d.Qualifiers())
		line := strings.Join(append(quals, "var", binding), " ")
		return symbol{"var", binding, []string{line}}, true
	}
	return symbol{}, false
}

func (ex *extractor) typeBody(name string, descriptor tree.Node) []string {
	switch td := descriptor.(type) {
	case *tree.RecordTypeDescriptorNode:
		openness := "open"
		if td.BodyStartDelimiter().Text() == "{|" {
			openness = "closed"
		}
		lines := []string{"type " + name + " record (" + openness + ")"}
		var fields []string
		for f := range items(td.Fields()) {
			fields = append(fields, ex.recordFieldLine(f))
		}
		if rest := td.RecordRestDescriptor(); rest != nil {
			fields = append(fields, "rest "+ex.text(rest.TypeName()))
		}
		sort.Strings(fields)
		return append(lines, fields...)
	case *tree.ObjectTypeDescriptorNode:
		header := strings.Join(append(qualTexts(td.ObjectTypeQualifiers()), "object"), " ")
		return append([]string{"type " + name + " " + header}, ex.objectMemberLines(td.Members())...)
	default:
		return []string{"type " + name + " = " + ex.text(descriptor)}
	}
}

func (ex *extractor) recordFieldLine(f tree.Node) string {
	switch fd := f.(type) {
	case *tree.RecordFieldNode:
		line := "field " + fd.FieldName().Text() + " " + ex.text(fd.TypeName())
		if fd.ReadonlyKeyword() != nil {
			line += " readonly"
		}
		if fd.QuestionMarkToken() != nil {
			line += " optional"
		}
		return line
	case *tree.RecordFieldWithDefaultValueNode:
		line := "field " + fd.FieldName().Text() + " " + ex.text(fd.TypeName())
		if fd.ReadonlyKeyword() != nil {
			line += " readonly"
		}
		return line + " = " + ex.text(fd.Expression())
	case *tree.TypeReferenceNode:
		return "include *" + ex.text(fd.TypeName())
	default:
		return "field? " + ex.text(f)
	}
}

func (ex *extractor) objectMemberLines(members tree.NodeList[tree.Node]) []string {
	var lines []string
	for m := range members.Iterator() {
		switch md := m.(type) {
		case *tree.ObjectFieldNode:
			if !isPublicToken(md.VisibilityQualifier()) {
				continue
			}
			line := "field " + md.FieldName().Text() + " " + ex.text(md.TypeName())
			if extra := qualTexts(md.QualifierList()); len(extra) > 0 {
				line += " " + strings.Join(extra, " ")
			}
			if expr := md.Expression(); expr != nil {
				line += " = " + ex.text(expr)
			}
			lines = append(lines, line)
		case *tree.FunctionDefinition:
			quals := qualTexts(md.QualifierList())
			if !callerVisibleMethod(quals, md.FunctionName().Text()) {
				continue
			}
			lines = append(lines, "method "+ex.fnLine(quals, md.FunctionName().Text(), ex.resourcePath(md.RelativeResourcePath()), md.FunctionSignature()))
		case *tree.MethodDeclarationNode:
			quals := qualTexts(md.QualifierList())
			if !callerVisibleMethod(quals, md.MethodName().Text()) {
				continue
			}
			lines = append(lines, "method "+ex.fnLine(quals, md.MethodName().Text(), ex.resourcePath(md.RelativeResourcePath()), md.MethodSignature()))
		case *tree.TypeReferenceNode:
			lines = append(lines, "include *"+ex.text(md.TypeName()))
		}
	}
	sort.Strings(lines)
	return lines
}

// A class/object method is part of the caller-facing contract when it is
// public, remote, or resource; `init` is included because callers invoke it
// via `new` even when it carries no visibility qualifier.
func callerVisibleMethod(quals []string, name string) bool {
	return contains(quals, "public") || contains(quals, "remote") || contains(quals, "resource") || name == "init"
}

func (ex *extractor) fnLine(quals []string, name string, resPath string, sig *tree.FunctionSignatureNode) string {
	var params []string
	for p := range items(sig.Parameters()) {
		params = append(params, ex.paramString(p))
	}
	line := strings.Join(append(quals, "function", name), " ")
	if resPath != "" {
		line += " " + resPath
	}
	line += "(" + strings.Join(params, ", ") + ")"
	if ret := sig.ReturnTypeDesc(); ret != nil {
		line += " returns " + ex.text(ret.Type())
	}
	return line
}

func (ex *extractor) paramString(p tree.ParameterNode) string {
	switch pd := p.(type) {
	case *tree.RequiredParameterNode:
		return ex.text(pd.TypeName()) + " " + tokenText(pd.ParamName())
	case *tree.DefaultableParameterNode:
		return ex.text(pd.TypeName()) + " " + tokenText(pd.ParamName()) + " = " + ex.text(pd.Expression())
	case *tree.RestParameterNode:
		return ex.text(pd.TypeName()) + "... " + tokenText(pd.ParamName())
	default:
		return ex.text(p)
	}
}

func (ex *extractor) resourcePath(path tree.NodeList[tree.Node]) string {
	var parts []string
	for p := range path.Iterator() {
		parts = append(parts, ex.text(p))
	}
	return strings.Join(parts, "")
}

func qualTexts(quals tree.NodeList[tree.Token]) []string {
	var out []string
	for q := range quals.Iterator() {
		out = append(out, q.Text())
	}
	sort.Strings(out)
	return out
}

func isPublicToken(t tree.Token) bool {
	return t != nil && t.Kind() == common.PUBLIC_KEYWORD
}

func tokenText(t tree.Token) string {
	if t == nil {
		return ""
	}
	return t.Text()
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func norm(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func printSorted(syms []symbol) {
	sort.Slice(syms, func(i, j int) bool {
		if syms[i].kind != syms[j].kind {
			return syms[i].kind < syms[j].kind
		}
		return syms[i].name < syms[j].name
	})
	for _, s := range syms {
		fmt.Printf("%s %s\n", s.kind, s.name)
		for _, l := range s.body {
			fmt.Println("  " + l)
		}
		fmt.Println()
	}
}
