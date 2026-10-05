package completion

import (
 "context"
 "fmt"
 "path/filepath"
 "testing"
 "github.com/ballerina-nutcracker/ballerina/ast"
 "github.com/ballerina-nutcracker/ballerina/ls/core/compile"
 "github.com/ballerina-nutcracker/ballerina/ls/core/event"
 "github.com/ballerina-nutcracker/ballerina/ls/core/workspace"
 "github.com/ballerina-nutcracker/ballerina/platform/palnative"
)
func TestTemporaryRestMappingProbe(t *testing.T) {
 text := "function foo(int a, int b = 0, int... rest) returns int { return a; }\nfunction main(int[] xs) { int x = foo(...xs); int y = foo(a = 1, ...xs); }\n"
 platform,cleanup:=palnative.NewPlatform();defer cleanup();bus:=event.New();defer bus.Close();projects:=workspace.New(platform,bus);compiler:=compile.New(projects,bus,compile.WithDebounce(0));defer compiler.Shutdown();root:=t.TempDir();_ = platform.FS.WriteFile(filepath.Join(root,"Ballerina.toml"),[]byte("[package]\norg = \"probe\"\nname = \"restprobe\"\nversion = \"0.1.0\"\n"));file:=filepath.Join(root,"main.bal");_ = platform.FS.WriteFile(file,[]byte(text));u,err:=workspace.NewFileURI("file://"+file);if err!=nil{t.Fatal(err)};_,err=projects.Apply(context.Background(),workspace.DocumentChange{Kind:workspace.ChangeOpen,URI:u,Text:text,Version:1,LanguageID:"ballerina"});if err!=nil{t.Fatal(err)};sm,ok:=compiler.SealedModuleFor(context.Background(),u);if !ok{t.Fatal("no sealed module")};fmt.Printf("stage=%v diagnostics=%d\n",sm.Stage(),len(sm.Context().Diagnostics()))
 ast.Walk(&restProbeVisitor{f:func(n ast.BLangNode){if c,ok:=n.(*ast.BLangInvocation);ok&&c.Name.GetValue()=="foo"{p:=c.GetPosition();fmt.Printf("call span=%d..%d args=%d\n",p.StartOffset(),p.EndOffset(),len(c.ArgExprs));for i,a:=range c.ArgExprs{if a==nil{fmt.Printf(" slot%d <nil>\n",i);continue};x:=a.(ast.BLangNode);q:=x.GetPosition();fmt.Printf(" slot%d %T %d..%d\n",i,x,q.StartOffset(),q.EndOffset())}}}},sm.PackageNode())
}
type restProbeVisitor struct{f func(ast.BLangNode)}
func(v *restProbeVisitor)Visit(n ast.BLangNode)ast.Visitor{if n==nil{return v};v.f(n);return v}
func(v *restProbeVisitor)VisitTypeData(*ast.TypeData)ast.Visitor{return v}
