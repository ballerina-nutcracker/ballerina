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

package palnative

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

type randomAccessFile struct {
	*os.File
	size int64
}

func (f *randomAccessFile) Size() int64 {
	return f.size
}

// OpenRandomAccess opens path for positional reads. Exposed at package level so
// test harnesses can wrap it with their own path mapping.
func OpenRandomAccess(path string) (pal.RandomAccessFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return &randomAccessFile{File: f, size: fi.Size()}, nil
}

// CreateSeekable opens path for writing; see pal.FS.CreateSeekable.
func CreateSeekable(path string, exclusive bool) (pal.SeekableFile, error) {
	flag := os.O_RDWR | os.O_CREATE
	if exclusive {
		flag |= os.O_EXCL
	} else {
		flag |= os.O_TRUNC
	}
	return os.OpenFile(path, flag, 0o666)
}

// RealPath follows Java's Path.toRealPath: the path is made absolute and every
// link along it resolved, and it must exist.
func RealPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}

// SameFile follows Java's Files.isSameFile: equal paths are the same file
// without the file system being asked.
func SameFile(first, second string) (bool, error) {
	if first == second {
		return true, nil
	}
	firstInfo, err := os.Stat(first)
	if err != nil {
		return false, err
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		return false, err
	}
	return os.SameFile(firstInfo, secondInfo), nil
}

// Chtimes sets the modification time of path and leaves its access time as it is.
func Chtimes(path string, modified time.Time) error {
	return os.Chtimes(path, time.Time{}, modified)
}
