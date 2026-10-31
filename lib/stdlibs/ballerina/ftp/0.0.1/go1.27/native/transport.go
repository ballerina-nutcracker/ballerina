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
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/textproto"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

var errNotExist = errors.New("file does not exist")

type remoteEntry struct {
	name       string
	isDir      bool
	isLink     bool
	size       int64
	modTime    time.Time
	readable   bool
	writable   bool
	executable bool
}

// session is the protocol-specific part of a connection. Paths are absolute paths on the
// server; resolving them against the client's root is done by the caller.
type session interface {
	stat(p string) (*remoteEntry, error)
	readDir(p string) ([]remoteEntry, error)
	open(p string) (io.ReadCloser, error)
	store(p string, r io.Reader, appendMode bool) error
	mkdir(p string) error
	removeFile(p string) error
	removeDir(p string) error
	rename(from, to string) error
	homeDir() (string, error)
	close() error
}

func dialSession(platform pal.Platform, ep *endpoint) (session, error) {
	if ep.protocol == protocolSFTP {
		return dialSFTP(platform, ep)
	}
	return dialFTP(platform, ep)
}

func (ep *endpoint) address() string {
	return net.JoinHostPort(ep.host, strconv.FormatInt(ep.port, 10))
}

func dialTCP(platform pal.Platform, ep *endpoint, address string) (net.Conn, error) {
	ctx := context.Background()
	if ep.connectTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ep.connectTimeout)
		defer cancel()
	}
	return platform.Net.Dial(ctx, "tcp", address, "", nil)
}

// idleTimeoutConn applies a read/write deadline before every I/O call, the Go analogue of a
// Java socket SO_TIMEOUT.
type idleTimeoutConn struct {
	net.Conn
	timeout time.Duration
	now     func() time.Time
}

func withIdleTimeout(conn net.Conn, timeout time.Duration, now func() time.Time) net.Conn {
	if timeout <= 0 {
		return conn
	}
	return &idleTimeoutConn{Conn: conn, timeout: timeout, now: now}
}

func (c *idleTimeoutConn) Read(b []byte) (int, error) {
	_ = c.SetReadDeadline(c.now().Add(c.timeout))
	return c.Conn.Read(b)
}

func (c *idleTimeoutConn) Write(b []byte) (int, error) {
	_ = c.SetWriteDeadline(c.now().Add(c.timeout))
	return c.Conn.Write(b)
}

// ftpSession pools control connections: a server connection carries one transfer at a time,
// so concurrent operations and open read streams each take their own connection, as VFS does.
type ftpSession struct {
	platform  pal.Platform
	ep        *endpoint
	tlsConfig *tls.Config
	mu        sync.Mutex
	idle      []*ftp.ServerConn
	closed    bool
	home      string
}

func dialFTP(platform pal.Platform, ep *endpoint) (session, error) {
	s := &ftpSession{platform: platform, ep: ep}
	if ep.protocol == protocolFTPS {
		cfg, err := platform.Net.ClientTLSConfig(ep.host, ep.tlsConfig)
		if err != nil {
			return nil, err
		}
		cfg.ClientSessionCache = tls.NewLRUClientSessionCache(0)
		s.tlsConfig = cfg
	}
	c, err := s.newConn()
	if err != nil {
		return nil, err
	}
	s.release(c)
	return s, nil
}

func (s *ftpSession) newConn() (*ftp.ServerConn, error) {
	control := true
	dial := func(network, address string) (net.Conn, error) {
		isControl := control
		control = false
		conn, err := dialTCP(s.platform, s.ep, address)
		if err != nil {
			return nil, err
		}
		timeout := s.ep.ftpDataTimeout
		if isControl {
			timeout = s.ep.ftpSocketTimeout
		}
		conn = withIdleTimeout(conn, timeout, s.platform.Time.Now)
		if s.tlsConfig == nil || (isControl && !s.ep.implicitTLS) {
			return conn, nil
		}
		tlsConn := tls.Client(conn, s.tlsConfig)
		if isControl {
			if err := tlsConn.Handshake(); err != nil {
				_ = conn.Close()
				return nil, err
			}
		}
		return tlsConn, nil
	}
	// commons-vfs lists directories with LIST, so MLSD is not used.
	opts := []ftp.DialOption{ftp.DialWithDialFunc(dial), ftp.DialWithDisabledMLSD(true)}
	if s.tlsConfig != nil {
		if s.ep.implicitTLS {
			opts = append(opts, ftp.DialWithTLS(s.tlsConfig))
		} else {
			opts = append(opts, ftp.DialWithExplicitTLS(s.tlsConfig))
		}
	}
	c, err := ftp.Dial(s.ep.address(), opts...)
	if err != nil {
		return nil, err
	}
	if err := c.Login(s.ep.username, s.ep.password); err != nil {
		_ = c.Quit()
		return nil, err
	}
	if s.ep.asciiMode {
		if err := c.Type(ftp.TransferTypeASCII); err != nil {
			_ = c.Quit()
			return nil, err
		}
	}
	return c, nil
}

// acquire returns a live idle connection, or a new one. An idle connection is checked with NOOP
// first, since the server may have dropped it while it sat in the pool.
func (s *ftpSession) acquire() (*ftp.ServerConn, error) {
	for {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return nil, errors.New("connection is closed")
		}
		n := len(s.idle)
		if n == 0 {
			s.mu.Unlock()
			return s.newConn()
		}
		c := s.idle[n-1]
		s.idle = s.idle[:n-1]
		s.mu.Unlock()
		if err := c.NoOp(); err == nil {
			return c, nil
		}
		_ = c.Quit()
	}
}

func (s *ftpSession) release(c *ftp.ServerConn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		_ = c.Quit()
		return
	}
	s.idle = append(s.idle, c)
}

// do runs fn on a pooled connection. A reply error leaves the connection usable; any other
// failure (I/O, timeout) discards it.
func (s *ftpSession) do(fn func(c *ftp.ServerConn) error) error {
	c, err := s.acquire()
	if err != nil {
		return err
	}
	err = fn(c)
	if err == nil || isReply(err) {
		s.release(c)
	} else {
		_ = c.Quit()
	}
	return err
}

func isReply(err error) bool {
	var reply *textproto.Error
	return errors.As(err, &reply)
}

func isNotFoundReply(err error) bool {
	var reply *textproto.Error
	return errors.As(err, &reply) && reply.Code == ftp.StatusFileUnavailable
}

// stat resolves an entry the way commons-vfs does over FTP, by listing its parent. The root and
// the login directory are known folders, so a jailed user never lists the ancestors of home.
func (s *ftpSession) stat(p string) (*remoteEntry, error) {
	if p == "/" || p == s.home {
		return &remoteEntry{name: path.Base(p), isDir: true, readable: true, writable: true}, nil
	}
	parent := path.Dir(p)
	if parent == p {
		return nil, errNotExist
	}
	entries, err := s.readDir(parent)
	if errors.Is(err, errNotExist) {
		return nil, errNotExist
	}
	if isReply(err) {
		// Servers disagree on how they answer LIST for a missing path, so a refusal means the
		// entry is missing only when the parent is.
		if _, parentErr := s.stat(parent); errors.Is(parentErr, errNotExist) {
			return nil, errNotExist
		}
	}
	if err != nil {
		return nil, err
	}
	name := path.Base(p)
	for i := range entries {
		if entries[i].name == name {
			return &entries[i], nil
		}
	}
	return nil, errNotExist
}

func (s *ftpSession) readDir(p string) ([]remoteEntry, error) {
	var list []*ftp.Entry
	linkIsDir := map[string]bool{}
	err := s.do(func(c *ftp.ServerConn) error {
		var err error
		if list, err = c.List(p); err != nil {
			return err
		}
		// A link takes its target's type, as in commons-vfs; CWD follows it on the server.
		for _, e := range list {
			if e.Type == ftp.EntryTypeLink {
				linkIsDir[e.Name] = c.ChangeDir(path.Join(p, path.Base(e.Name))) == nil
			}
		}
		return nil
	})
	if err != nil {
		if isNotFoundReply(err) {
			return nil, errNotExist
		}
		return nil, err
	}
	entries := make([]remoteEntry, 0, len(list))
	for _, e := range list {
		if e.Name == "." || e.Name == ".." {
			continue
		}
		entries = append(entries, remoteEntry{
			name:     path.Base(e.Name),
			isDir:    e.Type == ftp.EntryTypeFolder || linkIsDir[e.Name],
			isLink:   e.Type == ftp.EntryTypeLink,
			size:     int64(e.Size),
			modTime:  e.Time,
			readable: true,
			writable: true,
		})
	}
	return entries, nil
}

type ftpReader struct {
	resp    *ftp.Response
	session *ftpSession
	conn    *ftp.ServerConn
	once    sync.Once
}

func (r *ftpReader) Read(b []byte) (int, error) { return r.resp.Read(b) }

func (r *ftpReader) Close() error {
	var err error
	r.once.Do(func() {
		err = r.resp.Close()
		if err != nil {
			_ = r.conn.Quit()
			return
		}
		r.session.release(r.conn)
	})
	return err
}

func (s *ftpSession) open(p string) (io.ReadCloser, error) {
	c, err := s.acquire()
	if err != nil {
		return nil, err
	}
	resp, err := c.Retr(p)
	if err != nil {
		if isReply(err) {
			s.release(c)
		} else {
			_ = c.Quit()
		}
		return nil, err
	}
	return &ftpReader{resp: resp, session: s, conn: c}, nil
}

func (s *ftpSession) store(p string, r io.Reader, appendMode bool) error {
	return s.do(func(c *ftp.ServerConn) error {
		if appendMode {
			return c.Append(p, r)
		}
		return c.Stor(p, r)
	})
}

func (s *ftpSession) mkdir(p string) error {
	return s.do(func(c *ftp.ServerConn) error { return c.MakeDir(p) })
}

func (s *ftpSession) removeFile(p string) error {
	return s.do(func(c *ftp.ServerConn) error { return c.Delete(p) })
}

func (s *ftpSession) removeDir(p string) error {
	return s.do(func(c *ftp.ServerConn) error { return c.RemoveDir(p) })
}

func (s *ftpSession) rename(from, to string) error {
	return s.do(func(c *ftp.ServerConn) error { return c.Rename(from, to) })
}

func (s *ftpSession) homeDir() (string, error) {
	var dir string
	err := s.do(func(c *ftp.ServerConn) error {
		var err error
		dir, err = c.CurrentDir()
		return err
	})
	if err == nil {
		s.home = dir
	}
	return dir, err
}

func (s *ftpSession) close() error {
	s.mu.Lock()
	idle := s.idle
	s.idle = nil
	s.closed = true
	s.mu.Unlock()
	var errs []error
	for _, c := range idle {
		if err := c.Quit(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type sftpSession struct {
	ssh    *ssh.Client
	client *sftp.Client
}

func dialSFTP(platform pal.Platform, ep *endpoint) (session, error) {
	address := ep.address()
	conn, err := dialTCP(platform, ep, address)
	if err != nil {
		return nil, err
	}
	config, err := sshClientConfig(platform, ep)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if ep.connectTimeout > 0 {
		_ = conn.SetDeadline(platform.Time.Now().Add(ep.connectTimeout))
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	sshClient := ssh.NewClient(sshConn, chans, reqs)
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, err
	}
	return &sftpSession{ssh: sshClient, client: client}, nil
}

func sftpEntry(info os.FileInfo) remoteEntry {
	mode := info.Mode()
	return remoteEntry{
		name:       info.Name(),
		isDir:      info.IsDir(),
		size:       info.Size(),
		modTime:    info.ModTime(),
		readable:   mode&0o400 != 0,
		writable:   mode&0o200 != 0,
		executable: mode&0o100 != 0,
	}
}

func (s *sftpSession) stat(p string) (*remoteEntry, error) {
	info, err := s.client.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errNotExist
		}
		return nil, err
	}
	e := sftpEntry(info)
	e.name = path.Base(p)
	return &e, nil
}

func (s *sftpSession) readDir(p string) ([]remoteEntry, error) {
	infos, err := s.client.ReadDir(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errNotExist
		}
		return nil, err
	}
	entries := make([]remoteEntry, 0, len(infos))
	for _, info := range infos {
		entries = append(entries, sftpEntry(info))
	}
	return entries, nil
}

func (s *sftpSession) open(p string) (io.ReadCloser, error) {
	return s.client.Open(p)
}

func (s *sftpSession) store(p string, r io.Reader, appendMode bool) error {
	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if appendMode {
		flags = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	}
	f, err := s.client.OpenFile(p, flags)
	if err != nil {
		return err
	}
	if appendMode {
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			_ = f.Close()
			return err
		}
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func (s *sftpSession) mkdir(p string) error { return s.client.Mkdir(p) }

func (s *sftpSession) removeFile(p string) error { return s.client.Remove(p) }

func (s *sftpSession) removeDir(p string) error { return s.client.RemoveDirectory(p) }

func (s *sftpSession) rename(from, to string) error { return s.client.Rename(from, to) }

func (s *sftpSession) homeDir() (string, error) { return s.client.Getwd() }

func (s *sftpSession) close() error {
	err := s.client.Close()
	return errors.Join(err, s.ssh.Close())
}
