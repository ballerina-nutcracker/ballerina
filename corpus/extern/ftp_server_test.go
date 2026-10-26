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

package extern_test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	"github.com/pkg/sftp"
	"github.com/spf13/afero"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	ftpTestUser     = "ballerina"
	ftpTestPassword = "ballerina"
	sftpKeyPassword = "keypass"
)

// ftpFixture describes the servers started for one ftp test: an FTP server, an explicit and an
// implicit FTPS server and an SFTP server, each with its own in-memory file system.
type ftpFixture struct {
	ftpPort          int64
	ftpsPort         int64
	ftpsImplicitPort int64
	sftpPort         int64
	certPath         string
	keyPath          string
	encryptedKeyPath string
	knownHostsPath   string
	badKnownHosts    string
	hashedKnownHosts string
	missingKeyPath   string
}

type ftpTestDriver struct {
	fs       afero.Fs
	settings *ftpserver.Settings
	tls      *tls.Config
}

func (d *ftpTestDriver) GetSettings() (*ftpserver.Settings, error) { return d.settings, nil }

func (d *ftpTestDriver) ClientConnected(ftpserver.ClientContext) (string, error) {
	return "ballerina test server", nil
}

func (d *ftpTestDriver) ClientDisconnected(ftpserver.ClientContext) {}

func (d *ftpTestDriver) AuthUser(_ ftpserver.ClientContext, user, pass string) (ftpserver.ClientDriver, error) {
	if (user == ftpTestUser && pass == ftpTestPassword) || user == "anonymous" {
		return d.fs, nil
	}
	return nil, errors.New("invalid credentials")
}

func (d *ftpTestDriver) GetTLSConfig() (*tls.Config, error) {
	if d.tls == nil {
		return nil, errors.New("TLS is not configured")
	}
	return d.tls, nil
}

// listenAll binds every test server on the dual-stack wildcard address, as ftpserverlib binds its
// passive data ports: a more specific bind could otherwise be handed the same port number and
// capture another server's data connections.
func listenAll(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return ln
}

func startFTPServer(t *testing.T, tlsConfig *tls.Config, mode ftpserver.TLSRequirement) int64 {
	t.Helper()
	ln := listenAll(t)
	var err error
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	driver := &ftpTestDriver{
		fs:  afero.NewMemMapFs(),
		tls: tlsConfig,
		settings: &ftpserver.Settings{
			Listener:    ln,
			TLSRequired: mode,
			IdleTimeout: 30,
		},
	}
	if mode == ftpserver.ImplicitEncryption {
		driver.settings.Listener = tls.NewListener(ln, tlsConfig)
	}
	server := ftpserver.NewFtpServer(driver)
	server.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := server.Listen(); err != nil {
		t.Fatalf("ftp listen: %v", err)
	}
	go func() { _ = server.Serve() }()
	t.Cleanup(func() { _ = server.Stop() })
	return int64(ln.Addr().(*net.TCPAddr).Port)
}

func selfSignedTLS(t *testing.T, dir string) (*tls.Config, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(dir, "server.crt")
	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}, certPath
}

func writeClientKeys(t *testing.T, dir string) (ssh.PublicKey, string, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatal(err)
	}
	encrypted, err := ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte(sftpKeyPassword))
	if err != nil {
		t.Fatal(err)
	}
	encryptedPath := filepath.Join(dir, "id_ed25519_enc")
	if err := os.WriteFile(encryptedPath, pem.EncodeToMemory(encrypted), 0o600); err != nil {
		t.Fatal(err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	return sshPub, keyPath, encryptedPath
}

// startSFTPServer serves an in-memory SFTP file system over SSH with password and public key
// authentication. It returns the port and the server's host key.
func startSFTPServer(t *testing.T, clientKey ssh.PublicKey) (int64, ssh.PublicKey) {
	t.Helper()
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatal(err)
	}
	config := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == ftpTestUser && string(password) == ftpTestPassword {
				return nil, nil
			}
			return nil, errors.New("password rejected")
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() == ftpTestUser && string(key.Marshal()) == string(clientKey.Marshal()) {
				return nil, nil
			}
			return nil, errors.New("key rejected")
		},
	}
	config.AddHostKey(hostSigner)
	ln := listenAll(t)
	t.Cleanup(func() { _ = ln.Close() })
	handlers := sftp.InMemHandler()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveSFTP(conn, config, handlers)
		}
	}()
	return int64(ln.Addr().(*net.TCPAddr).Port), hostSigner.PublicKey()
}

func serveSFTP(conn net.Conn, config *ssh.ServerConfig, handlers sftp.Handlers) {
	_, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		_ = conn.Close()
		return
	}
	go ssh.DiscardRequests(reqs)
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unsupported channel")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go func() {
			for req := range requests {
				_ = req.Reply(req.Type == "subsystem" && string(req.Payload[4:]) == "sftp", nil)
			}
		}()
		go func() {
			server := sftp.NewRequestServer(channel, handlers)
			if err := server.Serve(); err == io.EOF {
				_ = server.Close()
			}
		}()
	}
}

func startFtpFixture(t *testing.T) *ftpFixture {
	t.Helper()
	dir := t.TempDir()
	tlsConfig, certPath := selfSignedTLS(t, dir)
	clientKey, keyPath, encryptedKeyPath := writeClientKeys(t, dir)
	sftpPort, hostKey := startSFTPServer(t, clientKey)
	knownHosts := filepath.Join(dir, "known_hosts")
	line := "[127.0.0.1]:" + strconv.FormatInt(sftpPort, 10) + " " + string(ssh.MarshalAuthorizedKey(hostKey))
	if err := os.WriteFile(knownHosts, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	_, otherKey, _ := ed25519.GenerateKey(rand.Reader)
	otherSigner, _ := ssh.NewSignerFromKey(otherKey)
	badKnownHosts := filepath.Join(dir, "known_hosts_bad")
	badLine := "[127.0.0.1]:" + strconv.FormatInt(sftpPort, 10) + " " + string(ssh.MarshalAuthorizedKey(otherSigner.PublicKey()))
	if err := os.WriteFile(badKnownHosts, []byte(badLine), 0o600); err != nil {
		t.Fatal(err)
	}
	hashedKnownHosts := filepath.Join(dir, "known_hosts_hashed")
	hashedLine := knownhosts.HashHostname("[127.0.0.1]:"+strconv.FormatInt(sftpPort, 10)) + " " +
		string(ssh.MarshalAuthorizedKey(hostKey))
	if err := os.WriteFile(hashedKnownHosts, []byte(hashedLine), 0o600); err != nil {
		t.Fatal(err)
	}
	return &ftpFixture{
		ftpPort:          startFTPServer(t, nil, ftpserver.ClearOrEncrypted),
		ftpsPort:         startFTPServer(t, tlsConfig, ftpserver.MandatoryEncryption),
		ftpsImplicitPort: startFTPServer(t, tlsConfig, ftpserver.ImplicitEncryption),
		sftpPort:         sftpPort,
		certPath:         certPath,
		keyPath:          keyPath,
		encryptedKeyPath: encryptedKeyPath,
		knownHostsPath:   knownHosts,
		badKnownHosts:    badKnownHosts,
		hashedKnownHosts: hashedKnownHosts,
		missingKeyPath:   filepath.Join(dir, "missing_key"),
	}
}
