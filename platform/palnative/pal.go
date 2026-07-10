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

// Package palnative provides the native-CLI implementation of pal.Platform.
// The HTTP factory and its TLS plumbing live in http.go; IO is small enough
// to inline here. Other environments (e.g. WASM/web-editor) supply their own
// pal.Platform without importing this package.
package palnative

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"ballerina-lang-go/platform/pal"
)

var processStart = time.Now()

// createParentDirs creates any missing ancestor directories for path, mirroring
// jBallerina's io module, which creates parent directories before opening a
// file for writing or appending.
func createParentDirs(path string) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0o755)
	}
	return nil
}

// NewPlatform returns the native-CLI pal.Platform, wiring os.Stdout/Stderr for
// IO and NewHTTPClient for HTTP. The returned cleanup function releases signal
// resources owned by the platform.
func NewPlatform() (pal.Platform, func()) {
	signals, cleanupSignals := newSignalSource()
	return pal.Platform{
		IO: pal.IO{
			Stdout: func(p []byte) (n int, err error) { return os.Stdout.Write(p) },
			Stderr: func(p []byte) (n int, err error) { return os.Stderr.Write(p) },
		},
		FS: pal.FS{
			ReadFile: func(path string) ([]byte, error) {
				return os.ReadFile(path)
			},
			WriteFile: func(path string, data []byte) error {
				if err := createParentDirs(path); err != nil {
					return err
				}
				return os.WriteFile(path, data, 0o644)
			},
			AppendFile: func(path string, data []byte) (err error) {
				if err := createParentDirs(path); err != nil {
					return err
				}
				f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
				if err != nil {
					return err
				}
				defer func() {
					if cerr := f.Close(); cerr != nil && err == nil {
						err = cerr
					}
				}()
				_, err = f.Write(data)
				return err
			},
			Getwd: os.Getwd,
			Mkdir: func(path string) error {
				return os.Mkdir(path, 0o755)
			},
			MkdirAll: func(path string) error {
				return os.MkdirAll(path, 0o755)
			},
			Remove:    os.Remove,
			RemoveAll: os.RemoveAll,
			Rename:    os.Rename,
			CreateFile: func(path string) error {
				f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
				if err != nil {
					return err
				}
				return f.Close()
			},
			Stat: func(path string) (*pal.FileInfo, error) {
				fi, err := os.Stat(path)
				if err != nil {
					return nil, err
				}
				absPath, _ := filepath.Abs(path)
				return &pal.FileInfo{
					AbsPath:    absPath,
					Size:       fi.Size(),
					ModifiedAt: fi.ModTime(),
					IsDir:      fi.IsDir(),
					IsSymlink:  false,
					IsReadable: IsReadable(path, fi),
					IsWritable: IsWritable(path, fi),
				}, nil
			},
			Lstat: func(path string) (*pal.FileInfo, error) {
				fi, err := os.Lstat(path)
				if err != nil {
					return nil, err
				}
				absPath, _ := filepath.Abs(path)
				return &pal.FileInfo{
					AbsPath:    absPath,
					Size:       fi.Size(),
					ModifiedAt: fi.ModTime(),
					IsDir:      fi.IsDir(),
					IsSymlink:  fi.Mode()&os.ModeSymlink != 0,
					IsReadable: IsReadable(path, fi),
					IsWritable: IsWritable(path, fi),
				}, nil
			},
			ReadDir: func(path string) ([]pal.FileInfo, error) {
				entries, err := os.ReadDir(path)
				if err != nil {
					return nil, err
				}
				result := make([]pal.FileInfo, 0, len(entries))
				for _, entry := range entries {
					childPath := filepath.Join(path, entry.Name())
					fi, err := entry.Info()
					if err != nil {
						continue
					}
					absPath, _ := filepath.Abs(childPath)
					result = append(result, pal.FileInfo{
						AbsPath:    absPath,
						Size:       fi.Size(),
						ModifiedAt: fi.ModTime(),
						IsDir:      fi.IsDir(),
						IsSymlink:  fi.Mode()&os.ModeSymlink != 0,
						IsReadable: IsReadable(childPath, fi),
						IsWritable: IsWritable(childPath, fi),
					})
				}
				return result, nil
			},
			Copy:          CopyFS,
			CreateTemp:    CreateTemp,
			CreateTempDir: CreateTempDir,
			Readlink:      os.Readlink,
			Watch:         Watch,
		},
		OS: pal.OS{
			GetEnv: func(name string) string {
				return os.Getenv(name)
			},
			GetUsername: func() string {
				u, err := user.Current()
				if err != nil {
					return ""
				}
				return u.Username
			},
			GetUserHome: func() string {
				home, err := os.UserHomeDir()
				if err != nil {
					return ""
				}
				return home
			},
			SetEnv: func(key, val string) error {
				return os.Setenv(key, val)
			},
			UnsetEnv: func(key string) error {
				return os.Unsetenv(key)
			},
			ListEnv: func() map[string]string {
				result := make(map[string]string)
				for _, e := range os.Environ() {
					for i := 0; i < len(e); i++ {
						if e[i] == '=' {
							result[e[:i]] = e[i+1:]
							break
						}
					}
				}
				return result
			},
			Exec: Exec,
		},
		Time: pal.Time{
			Now:          time.Now,
			MonotonicNow: func() time.Duration { return time.Since(processStart) },
			Sleep:        time.Sleep,
		},
		HTTP: pal.HTTP{
			NewClient: NewHTTPClient,
			Listen:    Listen,
		},
		Net: pal.Net{
			Dial:         Dial,
			Listen:       ListenTCP,
			DialPacket:   DialPacket,
			ListenPacket: ListenPacket,
		},
		Signals: signals,
	}, cleanupSignals
}

// Exec starts a subprocess and returns a handle to it. Exposed at package level
// so test harnesses can wire real subprocess execution into a pal.Platform.
func Exec(command string, args []string, envOverride map[string]string) (pal.ProcessHandle, error) {
	cmd := exec.Command(command, args...) //nolint:gosec
	if len(envOverride) > 0 {
		env := os.Environ()
		for k, v := range envOverride {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &nativeProcess{cmd: cmd, stdout: &stdout, stderr: &stderr}, nil
}

type nativeProcess struct {
	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func (p *nativeProcess) WaitForExit() (int, error) {
	err := p.cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}
	return 0, nil
}

func (p *nativeProcess) ReadStdout() ([]byte, error) {
	_ = p.cmd.Wait()
	return p.stdout.Bytes(), nil
}

func (p *nativeProcess) ReadStderr() ([]byte, error) {
	_ = p.cmd.Wait()
	return p.stderr.Bytes(), nil
}

func (p *nativeProcess) Kill() {
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

// FS helpers

func IsReadable(path string, _ os.FileInfo) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func IsWritable(path string, fi os.FileInfo) bool {
	if fi.IsDir() {
		return fi.Mode().Perm()&0o222 != 0
	}
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func CreateTemp(prefix, suffix, dir string) (string, error) {
	if dir == "" {
		dir = os.TempDir()
	}
	f, err := os.CreateTemp(dir, prefix+"*"+suffix)
	if err != nil {
		return "", err
	}
	name := f.Name()
	f.Close()
	abs, _ := filepath.Abs(name)
	return abs, nil
}

func CreateTempDir(prefix, suffix, dir string) (string, error) {
	if dir == "" {
		dir = os.TempDir()
	}
	path, err := os.MkdirTemp(dir, prefix+"*"+suffix)
	if err != nil {
		return "", err
	}
	return filepath.Abs(path)
}

func CopyFS(src, dst string, opts pal.CopyOptions) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if srcInfo.Mode()&os.ModeSymlink != 0 && opts.NoFollowLinks {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if opts.ReplaceExisting {
			os.Remove(dst) //nolint:errcheck
		}
		return os.Symlink(target, dst)
	}
	if srcInfo.IsDir() {
		return copyDir(src, dst, opts)
	}
	return copyFile(src, dst, opts)
}

func copyDir(src, dst string, opts pal.CopyOptions) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			if mkErr := os.MkdirAll(target, 0o755); mkErr != nil && !os.IsExist(mkErr) {
				return mkErr
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 && opts.NoFollowLinks {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if opts.ReplaceExisting {
				os.Remove(target) //nolint:errcheck
			}
			return os.Symlink(linkTarget, target)
		}
		return copyFile(path, target, opts)
	})
}

func copyFile(src, dst string, opts pal.CopyOptions) error {
	if !opts.ReplaceExisting {
		if _, err := os.Lstat(dst); err == nil {
			return &os.PathError{Op: "copy", Path: dst, Err: os.ErrExist}
		}
	}
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcF.Close()
	dstF, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstF.Close()
	if _, err := io.Copy(dstF, srcF); err != nil {
		return err
	}
	if opts.CopyAttributes {
		if info, err := os.Stat(src); err == nil {
			os.Chmod(dst, info.Mode()) //nolint:errcheck
		}
	}
	return nil
}
