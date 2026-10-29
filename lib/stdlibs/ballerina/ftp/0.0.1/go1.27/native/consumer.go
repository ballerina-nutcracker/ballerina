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

package native

import (
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

type addedFile struct {
	logical string
	entry   remoteEntry
}

type fileEvent struct {
	added   []addedFile
	deleted []string
}

// consumer polls one watched directory, following RemoteFileSystemConsumer: every file that
// passes the filters is reported as added on every poll, and a file is reported as deleted when
// it was reported before but is no longer listed. As there, processed gains a copy of a file's
// URI on every poll that reports it, so a deletion is reported once per such poll.
type consumer struct {
	platform  pal.Platform
	remote    *remoteClient
	watch     watchConfig
	mu        sync.Mutex
	processed []string
}

func (m *ftpModule) newConsumer(state *listenerState, watch watchConfig, failurePrefix string) (*consumer, error) {
	platform := m.rt.Platform()
	if watch.path == "" {
		watch.path = "/"
	}
	remote, err := connect(platform, state.ep)
	if err != nil {
		typed := asFtpError(err)
		return nil, &ftpError{kind: typed.kind, message: failurePrefix + typed.message}
	}
	entry, err := remote.stat(watch.path)
	if err != nil || entry == nil || !entry.isDir {
		_ = remote.close()
		return nil, newError(kindError, "FTP listener path \""+maskURLPassword(connectURL(state.ep, normalizeServicePath(watch.path)))+
			"\" does not refer to a folder on the remote file system.")
	}
	return &consumer{platform: platform, remote: remote, watch: watch}, nil
}

func (c *consumer) uri(logical string) string {
	return vfsURI(c.remote.ep, logical, false)
}

func (c *consumer) consume() (fileEvent, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	root := cleanPath(c.watch.path)
	children, err := c.remote.sess.readDir(c.remote.serverPath(root))
	if err != nil || len(children) == 0 {
		return fileEvent{}, false
	}
	rootFiles := make([]string, 0, len(children))
	for _, child := range children {
		if !child.isDir {
			rootFiles = append(rootFiles, child.name)
		}
	}
	var event fileEvent
	current := map[string]struct{}{}
	c.handleDirectory(root, children, rootFiles, &event, current)
	if len(c.processed) != len(current) {
		kept := c.processed[:0]
		for _, p := range c.processed {
			if _, listed := current[p]; listed {
				kept = append(kept, p)
			} else {
				event.deleted = append(event.deleted, p)
			}
		}
		c.processed = kept
	}
	return event, len(event.added) > 0 || len(event.deleted) > 0
}

func (c *consumer) handleDirectory(dir string, children []remoteEntry, rootFiles []string, event *fileEvent, current map[string]struct{}) {
	for _, child := range children {
		if c.watch.pattern != nil && !c.watch.pattern.MatchString(child.name) {
			continue
		}
		logical := path.Join(dir, child.name)
		if child.isDir {
			grandChildren, err := c.remote.sess.readDir(c.remote.serverPath(logical))
			if err == nil && len(grandChildren) > 0 {
				c.handleDirectory(logical, grandChildren, rootFiles, event, current)
			}
			continue
		}
		uri := c.uri(logical)
		current[uri] = struct{}{}
		if !c.passesAgeFilter(child) || !c.passesDependencies(child.name, rootFiles) {
			continue
		}
		event.added = append(event.added, addedFile{logical: logical, entry: child})
		c.processed = append(c.processed, uri)
	}
}

func (c *consumer) passesAgeFilter(entry remoteEntry) bool {
	f := c.watch.age
	if f == nil {
		return true
	}
	age := float64(c.platform.Time.Now().UnixMilli()-entry.modTime.UnixMilli()) / 1000.0
	if f.minAge != nil && age < *f.minAge {
		return false
	}
	return f.maxAge == nil || age <= *f.maxAge
}

var groupReference = regexp.MustCompile(`\$(\d+)`)

func (c *consumer) passesDependencies(name string, rootFiles []string) bool {
	for _, condition := range c.watch.dependencies {
		groups := condition.target.FindStringSubmatch(name)
		if groups == nil {
			continue
		}
		if !condition.satisfied(groups, rootFiles) {
			return false
		}
	}
	return true
}

func (d dependencyCondition) satisfied(groups []string, rootFiles []string) bool {
	matchedPatterns, totalMatches := 0, int64(0)
	for _, required := range d.requiredFiles {
		resolved := groupReference.ReplaceAllStringFunc(required, func(ref string) string {
			i, _ := strconv.Atoi(ref[1:])
			if i < len(groups) {
				return groups[i]
			}
			return ref
		})
		re, err := compilePattern(resolved)
		if err != nil {
			continue
		}
		matches := 0
		for _, f := range rootFiles {
			if re.MatchString(f) {
				matches++
			}
		}
		if matches > 0 {
			matchedPatterns++
		}
		totalMatches += int64(matches)
	}
	switch strings.ToUpper(d.mode) {
	case "ANY":
		return matchedPatterns > 0
	case "EXACT_COUNT":
		return totalMatches == d.requiredCount
	}
	return matchedPatterns == len(d.requiredFiles)
}
