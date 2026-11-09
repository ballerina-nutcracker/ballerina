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

package lib

import (
	"strings"
	"unsafe"
)

func b()                {}
func c(x int) int       { return x }
func d() int            { return 0 }
func two() (int, error) { return 0, nil }
func run(f func())      { f() } // want "function run only calls f;"
func id[T any](x T) T   { return x }

type value struct{}

func (value) m() int { return 0 }

type base struct{ x int }
type other struct{ y int }
type gbase[T any] struct{ v T }

func a()                          { b() }             // want "function a only calls b;"
func a2(x int) int                { return c(x) }     // want "function a2 only calls c;"
func a3(x int) int                { return c(x + 1) } // want "function a3 only calls c;"
func a4() int                     { return c(d()) }
func a5(v value) int              { return v.m() }                // want "function a5 only calls v.m;"
func a6(fn func())                { fn() }                        // want "function a6 only calls fn;"
func a7() string                  { return strings.ToUpper("x") } // want "function a7 only calls strings.ToUpper;"
func a8(x int) int                { return id[int](x) }           // want `function a8 only calls id\[int\];`
func a9() int                     { return (c(1)) }               // want "function a9 only calls c;"
func a10() (int, error)           { return two() }                // want "function a10 only calls two;"
func a11()                        { (b()) }                       // want "function a11 only calls b;"
func n1() []int                   { return make([]int, 0) }
func n2(e error)                  { panic(e) }
func n3(x int) int64              { return int64(x) }
func n4()                         { run(func() {}) }
func n5()                         { func() {}() }
func n6()                         { b(); b() }
func n7() int                     { return 1 }
func n8() (int, error)            { return c(1), nil }
func n9()                         { defer b() }
func n10()                        { go b() }
func n11()                        { _ = c(1) }
func n12()                        {}
func n13(x int) uintptr           { return unsafe.Sizeof(x) }
func n14(p *base) *base           { return (*base)(p) }
func n15(x gbase[int]) gbase[int] { return gbase[int](x) }
func r1()                         { r1() } // want "function r1 only calls r1;"
func ext()
func (value) m2() int { return c(1) }
func init()           { b() }
func _()              { b() }
func main()           { b() } // want "function main only calls b;"

func nestedLiteral() { run(identityCallback(func() {})) }

func identityCallback(f func()) func() { return f }

func constructValue() base          { return id(base{x: 1}) }
func constructPointer() *base       { return id(&base{x: 1}) }
func constructSlice() []int         { return id([]int{1}) }
func constructArray() [1]int        { return id([1]int{1}) }
func constructMap() map[string]int  { return id(map[string]int{"x": 1}) }
func nestedConstruction() int       { return c(id(base{x: 1}).x) }
func constructedReceiver() int      { return (value{}).m() }
func constructionStatement()        { consume(&base{x: 1}) }
func consume(*base)                 {}
func forwardExisting(p *base) *base { return id(p) } // want "function forwardExisting only calls id;"

// Nested evaluation does work in addition to the outer call.
func nestedCallStatement()            { c(d()) }
func nestedParenthesizedCall() int    { return c((d())) }
func nestedCallInIndex(xs []int) int  { return c(xs[d()]) }
func nestedConversion(x int64) int    { return c(int(x)) }
func nestedBuiltin(xs []int) int      { return c(len(xs)) }
func makeValue() value                { return value{} }
func calledReceiver() int             { return makeValue().m() }
func calledFunction() func()          { return b }
func calledCallee()                   { calledFunction()() }
func makeValues() []int               { return nil }
func calledIndexedCallee(fs []func()) { fs[d()]() }
func nestedSliceCall() int            { return c(makeValues()[0]) }
