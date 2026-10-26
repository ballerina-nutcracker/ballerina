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
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // the known_hosts hash format is defined over HMAC-SHA1
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/platform/pal"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func sshClientConfig(platform pal.Platform, ep *endpoint) (*ssh.ClientConfig, error) {
	var signer ssh.Signer
	if ep.privateKeyPath != "" {
		var err error
		if signer, err = loadPrivateKey(platform, ep); err != nil {
			return nil, err
		}
	}
	var auths []ssh.AuthMethod
	for _, method := range ep.preferredMethods {
		switch method {
		case "PUBLICKEY":
			if signer != nil {
				auths = append(auths, ssh.PublicKeys(signer))
			}
		case "PASSWORD":
			if ep.hasPassword {
				auths = append(auths, ssh.Password(ep.password))
			}
		case "KEYBOARD_INTERACTIVE":
			if ep.hasPassword {
				password := ep.password
				auths = append(auths, ssh.KeyboardInteractive(
					func(_, _ string, questions []string, _ []bool) ([]string, error) {
						answers := make([]string, len(questions))
						for i := range answers {
							answers[i] = password
						}
						return answers, nil
					}))
			}
		}
	}
	hostKeyCallback, err := hostKeyCallback(platform, ep)
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User:            ep.username,
		Auth:            auths,
		HostKeyCallback: hostKeyCallback,
		Timeout:         ep.connectTimeout,
	}, nil
}

func loadPrivateKey(platform pal.Platform, ep *endpoint) (ssh.Signer, error) {
	data, err := platform.FS.ReadFile(ep.privateKeyPath)
	if err != nil {
		return nil, newError(kindError, "SFTP private key file does not exist: "+ep.privateKeyPath)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, newError(kindError, "SFTP private key file is empty: "+ep.privateKeyPath)
	}
	var signer ssh.Signer
	if ep.privateKeyPassword != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(ep.privateKeyPassword))
	} else {
		signer, err = ssh.ParsePrivateKey(data)
	}
	if err != nil {
		return nil, fmt.Errorf("SFTP key authentication failed for private key \"%s\". %s Verify the key file is "+
			"a valid private key and the passphrase (if any) is correct.", ep.privateKeyPath, err.Error())
	}
	return signer, nil
}

// hostKeyCallback enables strict checking only when the known_hosts file exists, as JSch does
// with StrictHostKeyChecking; otherwise any server key is accepted.
func hostKeyCallback(platform pal.Platform, ep *endpoint) (ssh.HostKeyCallback, error) {
	if ep.knownHostsPath == "" {
		return ssh.InsecureIgnoreHostKey(), nil //nolint:gosec
	}
	knownHostsPath := ep.knownHostsPath
	if strings.HasPrefix(knownHostsPath, "~") {
		knownHostsPath = platform.OS.GetUserHome() + knownHostsPath[1:]
	}
	data, err := platform.FS.ReadFile(knownHostsPath)
	if err != nil {
		return ssh.InsecureIgnoreHostKey(), nil //nolint:gosec
	}
	entries, err := parseKnownHosts(data)
	if err != nil {
		return nil, err
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		candidates := []string{knownhosts.Normalize(hostname)}
		if remote != nil {
			candidates = append(candidates, knownhosts.Normalize(remote.String()))
		}
		for _, entry := range entries {
			if entry.matches(candidates) && bytes.Equal(entry.key.Marshal(), key.Marshal()) {
				return nil
			}
		}
		return fmt.Errorf("reject HostKey: %s", hostname)
	}, nil
}

type knownHost struct {
	patterns []string
	key      ssh.PublicKey
}

func parseKnownHosts(data []byte) ([]knownHost, error) {
	var entries []knownHost
	rest := data
	for len(rest) > 0 {
		marker, hosts, key, _, next, err := ssh.ParseKnownHosts(rest)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("invalid known_hosts file: %s", err.Error())
		}
		rest = next
		if marker == "" {
			entries = append(entries, knownHost{patterns: hosts, key: key})
		}
	}
	return entries, nil
}

func (h knownHost) matches(candidates []string) bool {
	for _, pattern := range h.patterns {
		for _, candidate := range candidates {
			if strings.HasPrefix(pattern, "|1|") {
				if hashedMatch(pattern, candidate) {
					return true
				}
				continue
			}
			if pattern == candidate {
				return true
			}
		}
	}
	return false
}

// hashedMatch checks a hashed known_hosts pattern `|1|salt|hash` against a host.
func hashedMatch(pattern, host string) bool {
	parts := strings.Split(pattern, "|")
	if len(parts) != 4 {
		return false
	}
	salt, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	mac := hmac.New(sha1.New, salt)
	mac.Write([]byte(host))
	return hmac.Equal(mac.Sum(nil), want)
}
