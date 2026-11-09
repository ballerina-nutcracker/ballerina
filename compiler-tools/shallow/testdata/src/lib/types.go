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

type s1 struct{ base }  // want "struct s1 only embeds base;"
type s2 struct{ *base } // want `struct s2 only embeds \*base;`
type s3 struct{ base }

func (s3) m() {}

type s4 struct{ base }

func (*s4) m() {}

type s5 struct{ base }
type s6 struct{ base } // want "struct s6 only embeds base;"
type s7 struct{ b base }
type s8 struct {
	base
	other
}
type s9 struct {
	base
	y int
}
type s10 struct{ io.Reader }
type (
	s11 struct{ base } // want "struct s11 only embeds base;"
)
type s12 struct{ base } // want "struct s12 only embeds base;"
type s13 struct{ base }
type s13a = s13

func (s13a) m() {}

type s14 struct{ base }
type s14p = *s14

func (s14p) m() {}

type s15 struct{ base }
type s15a = s15

func (*s15a) m() {}

type ba = base
type s16 struct{ ba } // want "struct s16 only embeds ba;"
type s21 struct{ base }
type s23 struct{ base }           // want "struct s23 only embeds base;"
type g1[T any] struct{ gbase[T] } // want `struct g1 only embeds gbase\[T\];`
type g2[T any] struct{ gbase[T] }

func (g2[T]) m() {}

type g3[K, V any] struct{ gbase[K] }

func (*g3[K, V]) m() {}

type g4[K, V any] struct{ gbase[K] }
type g5[T any] struct{ gbase[T] }
type a1 = base

type i1 interface{ io.Reader } // want "interface i1 only embeds io.Reader;"
type i2 interface {
	io.Reader
	Close()
}
type i3 interface {
	io.Reader
	io.Writer
}
type i4 interface{ comparable } // want "interface i4 only embeds comparable;"
type i5 interface{ any }        // want "interface i5 only embeds any;"
type c1 interface{ ~int | ~string }
type c2 interface{ int }
type c3 interface{ ~int }
