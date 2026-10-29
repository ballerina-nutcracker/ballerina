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
	"bytes"
	"errors"
	"io"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
)

var (
	urlSchemePattern        = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://`)
	userInfoPasswordPattern = regexp.MustCompile(`://([^/@:]+):([^/@]*)@`)
)

// maskURLPassword mirrors FileTransportUtils.maskUrlPassword: only a message that starts with a
// URL has its first password replaced.
func maskURLPassword(s string) string {
	if !urlSchemePattern.MatchString(s) {
		return s
	}
	loc := userInfoPasswordPattern.FindStringSubmatchIndex(s)
	if loc == nil {
		return s
	}
	return s[:loc[0]] + "://" + s[loc[2]:loc[3]] + ":***@" + s[loc[1]:]
}

func formatInt(i int64) string {
	return strconv.FormatInt(i, 10)
}

// cleanPath resolves a client path against the root, as VFS resolveFile does.
func cleanPath(p string) string {
	return path.Clean("/" + p)
}

// connectURL renders the URL FtpUtil.createUrl builds: the port is kept even when it is the
// default, and an empty password still produces `user:`.
func connectURL(ep *endpoint, p string) string {
	userInfo := url.User(ep.username)
	if ep.hasPassword {
		userInfo = url.UserPassword(ep.username, ep.password)
	}
	host := ep.host + ":" + formatInt(ep.port)
	u := url.URL{Scheme: ep.protocol, User: userInfo, Host: host, Path: p}
	s := u.String()
	if ep.hasPassword && ep.password == "" {
		s = strings.Replace(s, "://"+userInfo.String()+"@", "://"+userInfo.String()+":@", 1)
	}
	return s
}

// vfsURI renders a file URI the way VFS GenericFileName does: the default port is dropped and
// the path escapes only `%`, space and `#`. Only a masked URI names the user, as `user:***@`
// when there is a password; the others carry no credentials, where jBallerina exposes the
// password (ballerina-library#9381).
func vfsURI(ep *endpoint, p string, masked bool) string {
	return vfsRootURI(ep, masked) + vfsPathEscaper.Replace(strings.TrimPrefix(p, "/"))
}

var vfsPathEscaper = strings.NewReplacer("%", "%25", " ", "%20", "#", "%23")

func vfsRootURI(ep *endpoint, masked bool) string {
	var b strings.Builder
	b.WriteString(ep.protocol)
	b.WriteString("://")
	if masked {
		b.WriteString(url.User(ep.username).String())
		if ep.hasPassword && ep.password != "" {
			b.WriteString(":***")
		}
		b.WriteByte('@')
	}
	b.WriteString(ep.host)
	if ep.port != defaultPort(ep.protocol) {
		b.WriteByte(':')
		b.WriteString(formatInt(ep.port))
	}
	b.WriteByte('/')
	return b.String()
}

// remoteClient implements the file operations of an ftp:Client on top of a session, adding
// the commons-vfs behaviours jBallerina relies on (parent creation, copy).
type remoteClient struct {
	platform pal.Platform
	ep       *endpoint
	sess     session
	home     string
}

func connect(platform pal.Platform, ep *endpoint) (*remoteClient, error) {
	sess, err := dialSession(platform, ep)
	if err != nil {
		var typed *ftpError
		if errors.As(err, &typed) {
			return nil, typed
		}
		message := "Error while connecting to the FTP server with URL: " + maskURLPassword(connectURL(ep, "/")) +
			". " + err.Error()
		kind := kindConnection
		if code, ok := ftpReplyCode(message, err); ok && isServiceUnavailableCode(code) {
			kind = kindServiceUnavailable
		}
		return nil, &ftpError{kind: kind, message: message}
	}
	c := &remoteClient{platform: platform, ep: ep, sess: sess}
	if ep.userDirIsRoot {
		home, err := sess.homeDir()
		if err != nil {
			_ = sess.close()
			return nil, operationError("Error while connecting to the FTP server with URL: "+
				maskURLPassword(connectURL(ep, "/"))+". "+err.Error(), nil)
		}
		if home != "/" {
			c.home = home
		}
	}
	return c, nil
}

// serverPath maps a client path to the path on the server.
func (c *remoteClient) serverPath(p string) string {
	clean := cleanPath(p)
	if c.home == "" {
		return clean
	}
	return path.Join(c.home, clean)
}

func (c *remoteClient) uri(p string) string {
	return vfsURI(c.ep, cleanPath(p), true)
}

func (c *remoteClient) stat(p string) (*remoteEntry, error) {
	entry, err := c.sess.stat(c.serverPath(p))
	if errors.Is(err, errNotExist) {
		return nil, nil
	}
	return entry, err
}

func (c *remoteClient) statOrFail(p, operation string) (*remoteEntry, error) {
	entry, err := c.stat(p)
	if err != nil {
		return nil, operationError("Could not determine the type of file \""+c.uri(p)+"\".", err)
	}
	if entry == nil {
		return nil, newError(kindFileNotFound, "Failed to "+operation+": "+c.uri(p)+" not found")
	}
	return entry, nil
}

func (c *remoteClient) openFile(p string) (io.ReadCloser, error) {
	return c.open(p, "Could not read from")
}

// open opens a file for reading; a folder fails with notFilePrefix, which differs between
// streamed and whole-file reads in jBallerina.
func (c *remoteClient) open(p, notFilePrefix string) (io.ReadCloser, error) {
	entry, err := c.statOrFail(p, "read file")
	if err != nil {
		return nil, err
	}
	if entry.isDir {
		return nil, newError(kindError, notFilePrefix+" \""+c.uri(p)+"\" because it is not a file.")
	}
	r, err := c.sess.open(c.serverPath(p))
	if err != nil {
		return nil, operationError("Could not read file \""+c.uri(p)+"\".", err)
	}
	return r, nil
}

func (c *remoteClient) readAll(p string) ([]byte, error) {
	r, err := c.open(p, "Could not determine the size of")
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(r)
	closeErr := r.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, operationError("Could not read file \""+c.uri(p)+"\".", err)
	}
	return data, nil
}

func (c *remoteClient) mkdirAll(serverPath string) error {
	entry, err := c.sess.stat(serverPath)
	if err == nil {
		if entry.isDir {
			return nil
		}
		return errors.New("a file exists at " + serverPath)
	}
	if !errors.Is(err, errNotExist) {
		return err
	}
	if parent := path.Dir(serverPath); parent != serverPath {
		if err := c.mkdirAll(parent); err != nil {
			return err
		}
	}
	return c.sess.mkdir(serverPath)
}

func (c *remoteClient) write(p string, r io.Reader, appendMode bool) error {
	target := c.serverPath(p)
	if err := c.mkdirAll(path.Dir(target)); err != nil {
		return operationError("Could not create folder \""+c.uri(path.Dir(cleanPath(p)))+"\".", err)
	}
	if err := c.sess.store(target, r, appendMode); err != nil {
		return operationError("Could not write to \""+c.uri(p)+"\".", err)
	}
	return nil
}

func (c *remoteClient) writeBytes(p string, data []byte, appendMode bool) error {
	return c.write(p, bytes.NewReader(data), appendMode)
}

func (c *remoteClient) delete(p string) error {
	return c.remove(p, "delete file")
}

// rmdir removes a folder only when it is empty, as delete does; jBallerina removes a non-empty
// folder recursively, which spec section 3.4 forbids (ballerina-library#9410).
func (c *remoteClient) rmdir(p string) error {
	return c.remove(p, "delete directory")
}

func (c *remoteClient) remove(p string, operation string) error {
	entry, err := c.statOrFail(p, operation)
	if err != nil {
		return err
	}
	target := c.serverPath(p)
	if entry.isDir {
		var children []remoteEntry
		if children, err = c.sess.readDir(target); err == nil && len(children) > 0 {
			return newError(kindError, "Could not delete \""+c.uri(p)+"\".")
		}
		if err == nil {
			err = c.sess.removeDir(target)
		}
	} else {
		err = c.sess.removeFile(target)
	}
	if err != nil {
		return operationError("Could not delete \""+c.uri(p)+"\".", err)
	}
	return nil
}

func (c *remoteClient) mkdir(p string) error {
	entry, err := c.stat(p)
	if err != nil {
		return operationError("Could not create folder \""+c.uri(p)+"\".", err)
	}
	if entry != nil {
		return newError(kindFileAlreadyExists, "Directory exists: "+c.uri(p))
	}
	if err := c.mkdirAll(c.serverPath(p)); err != nil {
		return operationError("Could not create folder \""+c.uri(p)+"\".", err)
	}
	return nil
}

func (c *remoteClient) prepareDestination(dest, existsMessage string) error {
	entry, err := c.stat(dest)
	if err != nil {
		return operationError("Could not determine the type of file \""+c.uri(dest)+"\".", err)
	}
	if entry != nil {
		return newError(kindFileAlreadyExists, "The file at "+c.uri(dest)+existsMessage)
	}
	parent := path.Dir(c.serverPath(dest))
	if err := c.mkdirAll(parent); err != nil {
		return operationError("Could not create folder \""+c.uri(path.Dir(cleanPath(dest)))+"\".", err)
	}
	return nil
}

func (c *remoteClient) rename(src, dest string) error {
	if _, err := c.statOrFail(src, "rename file"); err != nil {
		return err
	}
	if err := c.prepareDestination(dest, " already exists or it is a directory"); err != nil {
		return err
	}
	if err := c.sess.rename(c.serverPath(src), c.serverPath(dest)); err != nil {
		return operationError("Could not rename \""+c.uri(src)+"\" to \""+c.uri(dest)+"\".", err)
	}
	return nil
}

func (c *remoteClient) copy(src, dest string) error {
	entry, err := c.statOrFail(src, "copy file")
	if err != nil {
		return err
	}
	if err := c.prepareDestination(dest, " already exists"); err != nil {
		return err
	}
	if entry.isDir {
		if err := c.sess.mkdir(c.serverPath(dest)); err != nil {
			return operationError("Could not copy \""+c.uri(src)+"\" to \""+c.uri(dest)+"\".", err)
		}
		return nil
	}
	r, err := c.sess.open(c.serverPath(src))
	if err != nil {
		return operationError("Could not copy \""+c.uri(src)+"\" to \""+c.uri(dest)+"\".", err)
	}
	storeErr := c.sess.store(c.serverPath(dest), r, false)
	if err := errors.Join(storeErr, r.Close()); err != nil {
		return operationError("Could not copy \""+c.uri(src)+"\" to \""+c.uri(dest)+"\".", err)
	}
	return nil
}

func (c *remoteClient) exists(p string) (bool, error) {
	entry, err := c.stat(p)
	if err != nil {
		return false, operationError("Could not determine the type of file \""+c.uri(p)+"\".", err)
	}
	return entry != nil, nil
}

func (c *remoteClient) size(p string) (int64, error) {
	entry, err := c.stat(p)
	if err != nil {
		return 0, operationError("Could not determine the size of \""+c.uri(p)+"\".", err)
	}
	if entry == nil || entry.isDir {
		return 0, newError(kindError, "Could not determine the size of \""+c.uri(p)+"\" because it is not a file.")
	}
	return int64(int32(entry.size)), nil
}

func (c *remoteClient) isDirectory(p string) (bool, error) {
	entry, err := c.stat(p)
	if err != nil {
		return false, operationError("Could not determine the type of file \""+c.uri(p)+"\".", err)
	}
	if entry == nil {
		return false, newError(kindFileNotFound, p+" does not exist to check if it is a directory.")
	}
	return entry.isDir, nil
}

func (c *remoteClient) list(p string) ([]remoteEntry, error) {
	entry, err := c.stat(p)
	if err != nil {
		return nil, operationError("Could not list the contents of \""+c.uri(p)+"\".", err)
	}
	if entry == nil || !entry.isDir {
		return nil, newError(kindError, "Could not list the contents of \""+c.uri(p)+"\" because it is not a folder.")
	}
	entries, err := c.sess.readDir(c.serverPath(p))
	if err != nil {
		return nil, operationError("Could not list the contents of \""+c.uri(p)+"\".", err)
	}
	return entries, nil
}

func (c *remoteClient) close() error {
	return c.sess.close()
}
