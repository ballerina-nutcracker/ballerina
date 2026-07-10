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
	"io/fs"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"

	"ballerina-lang-go/platform/pal"
)

// Watch monitors path (and, if recursive, every subdirectory beneath it) for
// filesystem changes using the OS's native file-event mechanism (inotify,
// kqueue, or ReadDirectoryChangesW, via fsnotify). Newly created
// subdirectories are registered dynamically when recursive is true, matching
// the behaviour of jBallerina's WatchService-backed directory listener.
func Watch(path string, recursive bool, handler pal.WatchHandler) (pal.WatchHandle, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := addWatchDirs(w, path, recursive); err != nil {
		_ = w.Close()
		return nil, err
	}
	go dispatchWatchEvents(w, recursive, handler)
	return w, nil
}

func dispatchWatchEvents(w *fsnotify.Watcher, recursive bool, handler pal.WatchHandler) {
	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return
			}
			if recursive && event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = addWatchDirs(w, event.Name, true)
				}
			}
			if op, ok := translateWatchOp(event.Op); ok {
				handler(pal.WatchEvent{Path: event.Name, Op: op})
			}
		case _, ok := <-w.Errors:
			if !ok {
				return
			}
		}
	}
}

// addWatchDirs registers root (and, if recursive, every directory beneath
// it) with w. fsnotify has no built-in recursive mode; each directory must
// be registered individually.
func addWatchDirs(w *fsnotify.Watcher, root string, recursive bool) error {
	if !recursive {
		return w.Add(root)
	}
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.Add(p)
		}
		return nil
	})
}

// translateWatchOp maps an fsnotify op to a pal.WatchOp. A rename is
// reported as a delete of the old path — the new path arrives as a separate
// Create event on the destination watch, mirroring how Java's WatchService
// (used by jBallerina's directory listener) reports a same-directory move as
// a delete-then-create pair.
func translateWatchOp(op fsnotify.Op) (pal.WatchOp, bool) {
	switch {
	case op&fsnotify.Create != 0:
		return pal.WatchCreate, true
	case op&fsnotify.Write != 0:
		return pal.WatchModify, true
	case op&fsnotify.Remove != 0 || op&fsnotify.Rename != 0:
		return pal.WatchDelete, true
	default:
		return 0, false
	}
}
