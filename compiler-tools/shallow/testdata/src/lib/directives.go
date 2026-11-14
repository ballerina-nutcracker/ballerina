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

import "io"

//shallow:allow keeps the public name stable
func e1() { b() }

//shallow:allow distinct type for switches
type e2 struct{ base }

type (
	//shallow:allow reason
	e3 struct{ base }
)

/* want "requires a reason" */ //shallow:allow
func e4()                      { b() }

/* want "e5 is not shallow; remove the unnecessary //shallow:allow directive" */ //shallow:allow reason
func e5() int                                                                    { return 1 }

/* want "m3 is not shallow; remove the unnecessary //shallow:allow directive" */ //shallow:allow reason
func (value) m3() int                                                            { return c(1) }

/* want "e6 is not shallow; remove the unnecessary //shallow:allow directive" */ //shallow:allow reason
type e6 struct{ x int }

/* want "must be in the doc comment" */ //shallow:allow reason
type (
	e7 struct{ base } // want "struct e7 only embeds base;"
)

//shallow:allow first
/* want "duplicate //shallow:allow directive" */ //shallow:allow second
func e8()                                        { b() }

/* want "requires a reason" */ //shallow:allow
func e9() int                  { return 1 }

/* want "e10 is not shallow; remove the unnecessary //shallow:allow directive" */ //shallow:allow reason
type e10 struct{ base }

type ( /* want "must be in the doc comment" */ //shallow:allow reason
	e11 struct{ base } // want "struct e11 only embeds base;"
)

func e12() { b() } /* want "function e12 only calls b;" "must be in the doc comment" */ //shallow:allow reason

//shallow:allowx reason
//shallow:allowed x
func e13() { b() } // want "function e13 only calls b;"

//shallow:allow first
/* want "duplicate //shallow:allow directive" */ //shallow:allow
func e14()                                       { b() }

//shallow:allow	reason
func e15() { b() }

/* want "requires a reason" */ //shallow:allow
/* want "duplicate //shallow:allow directive" */ //shallow:allow second
func e16()                                       { b() }

/* want "e17 is not shallow; remove the unnecessary //shallow:allow directive" */ //shallow:allow first
/* want "duplicate //shallow:allow directive" */ //shallow:allow second
func e17() int                                   { return 1 }

func f1() {
	/* want "must be in the doc comment" */ //shallow:allow reason
	b()
	b()
}
func f2() {
	type l1 struct{ base }
	/* want "must be in the doc comment" */ //shallow:allow reason
	type l2 struct{ base }
}

/* want "must be in the doc comment" */ //shallow:allow reason
var v1 = 1

/* want "must be in the doc comment" */ //shallow:allow
var v2 = 2

type fieldDirective struct {
	/* want "must be in the doc comment" */ //shallow:allow reason
	x int
}

/* want "must be in the doc comment" */ //shallow:allow reason

/* want "definedInt is not shallow; remove the unnecessary //shallow:allow directive" */ //shallow:allow a distinct scalar type
type definedInt int

//shallow:allow keeps the public interface name stable
type allowedInterface interface{ io.Reader }

/*shallow:allow block comments are not directives*/
func blockComment() { b() } // want "function blockComment only calls b;"

/* want "constructed is not shallow; remove the unnecessary //shallow:allow directive" */ //shallow:allow constructor
func constructed() base                                                                   { return id(base{x: 1}) }
