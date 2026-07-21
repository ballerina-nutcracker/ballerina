// Copyright (c) 2025, WSO2 LLC. (http://www.wso2.com).
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

package centralclient

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"ballerina/common/bfs"

	"github.com/Masterminds/semver/v3"
)

const (
	DeprecatedMetaFileName = "deprecated.txt"
)

var (
	SetBallerinaStageCentral = os.Getenv(BallerinaStageCentral) == "true"
	SetBallerinaDevCentral   = os.Getenv(BallerinaDevCentral) == "true"
	SetTestModeActive        = os.Getenv(TestModeActive) == "true"
)

func createBalaInHomeRepo(balaDownloadResponse *http.Response, fsys fs.FS, pkgPathInBalaCache, pkgOrg, pkgName string, isNightlyBuild bool, deprecationMsg, newUrl, contentDisposition string, trueDigest string, clientContext ClientContext) error {
	responseContentLength := balaDownloadResponse.ContentLength
	if responseContentLength <= 0 {
		return NewCentralClientError(clientContext.formatLog("invalid response from the server, please try again!"))
	}

	resolvedURI := balaDownloadResponse.Header.Get(ResolvedRequestedURI)
	if resolvedURI == "" {
		resolvedURI = newUrl
	}

	uriParts := strings.Split(resolvedURI, "/")
	pkgVersion := uriParts[len(uriParts)-2]

	validPkgVersion, err := validatePackageVersion(pkgVersion, clientContext)
	if err != nil {
		return err
	}

	balaFile := getBalaFileName(contentDisposition, uriParts[len(uriParts)-1])
	platform := getPlatformFromBala(balaFile, pkgName, pkgVersion)

	// <user.home>.ballerina/bala_cache/<org-name>/<pkg-name>/<pkg-version>
	balaCacheWithPkgPath := path.Join(pkgPathInBalaCache, validPkgVersion, platform)

	info, err := fs.Stat(fsys, balaCacheWithPkgPath)
	if err == nil && info.IsDir() {
		entries, err := fs.ReadDir(fsys, balaCacheWithPkgPath)
		if err != nil {
			return NewPackageAlreadyExistsError(clientContext.formatLog(fmt.Sprintf("error accessing bala : %s", balaCacheWithPkgPath)), validPkgVersion)
		}

		if len(entries) > 0 {
			deprecatedFilePath := path.Join(balaCacheWithPkgPath, DeprecatedMetaFileName)
			if _, err := fs.Stat(fsys, deprecatedFilePath); err == nil && deprecationMsg == "" {
				if err := bfs.Remove(fsys, deprecatedFilePath); err != nil {
					return NewPackageAlreadyExistsError(clientContext.formatLog(fmt.Sprintf("error accessing bala : %s", balaCacheWithPkgPath)), validPkgVersion)
				}
			} else if deprecationMsg != "" {
				if err := bfs.WriteFile(fsys, deprecatedFilePath, []byte(deprecationMsg), 0o644); err != nil {
					return NewPackageAlreadyExistsError(clientContext.formatLog(fmt.Sprintf("error accessing bala : %s", balaCacheWithPkgPath)), validPkgVersion)
				}
			}

			return NewPackageAlreadyExistsError(clientContext.formatLog(fmt.Sprintf("package already exists in the home repository: %s", balaCacheWithPkgPath)), validPkgVersion)
		}
	}

	// Create the following temp path
	// bala/<org-name>/<pkg-name>/<pkg-version_temp/<platform>
	tempPath := path.Join(pkgPathInBalaCache, fmt.Sprintf("%s_temp", validPkgVersion), platform)
	if err := createBalaFileDirectory(fsys, tempPath, clientContext); err != nil {
		return err
	}

	if err := writeBalaFile(balaDownloadResponse, fsys, path.Join(tempPath, balaFile), fmt.Sprintf("%s/%s:%s", pkgOrg, pkgName, validPkgVersion), trueDigest, clientContext); err != nil {
		return err
	}

	tempDir := path.Dir(tempPath)
	platformDir := path.Dir(balaCacheWithPkgPath)

	if err := bfs.Move(fsys, tempDir, platformDir); err != nil {
		return NewCentralClientError(clientContext.formatLog("error creating directory for bala file"))
	}

	if err := handleNightlyBuild(isNightlyBuild, fsys, balaCacheWithPkgPath, clientContext); err != nil {
		return err
	}

	if err := handlePackageDeprecation(deprecationMsg, fsys, balaCacheWithPkgPath, clientContext); err != nil {
		return err
	}

	return nil
}

func validatePackageVersion(pkgVersion string, clientContext ClientContext) (string, error) {
	if pkgVersion == "" {
		return "", NewCentralClientError(clientContext.formatLog("Version cannot be empty"))
	}

	version, err := semver.StrictNewVersion(pkgVersion)
	if err != nil {
		return "", NewCentralClientError(clientContext.formatLog(fmt.Sprintf("Invalid version: '%s'. %s", pkgVersion, err.Error())))
	}

	return version.String(), nil
}

func getBalaFileName(contentDisposition, balaFile string) string {
	if contentDisposition != "" {
		prefix := "attachment; filename="
		if strings.HasPrefix(contentDisposition, prefix) {
			return contentDisposition[len(prefix):]
		}
	}

	return balaFile
}

func getPlatformFromBala(balaName, packageName, version string) string {
	parts := strings.SplitN(balaName, packageName+"-", 2)
	if len(parts) < 2 {
		return ""
	}
	parts = strings.SplitN(parts[1], "-"+version, 2)
	return parts[0]
}

func createBalaFileDirectory(fsys fs.FS, fullPathToStoreBala string, clientContext ClientContext) error {
	if err := bfs.MkdirAll(fsys, fullPathToStoreBala, 0o755); err != nil {
		return NewCentralClientError(clientContext.formatLog("error creating directory for bala file"))
	}
	return nil
}

func writeBalaFile(balaDownloadResponse *http.Response, fsys fs.FS, balaPath, fullPkgName string, trueDigest string, clientContext ClientContext) error {
	balaDownloadResponseBody := balaDownloadResponse.Body

	if balaDownloadResponseBody == nil {
		return NewCentralClientError(clientContext.formatLog(fmt.Sprintf("error occurred extracting bytes of bala file: %s", fullPkgName)))
	}

	if err := writeAndHandleProgress(balaDownloadResponseBody, balaDownloadResponse.ContentLength, fsys, balaPath, clientContext); err != nil {
		return err
	}

	if err := extractBala(fsys, balaPath, path.Dir(balaPath), trueDigest, fullPkgName, clientContext); err != nil {
		return NewCentralClientError(clientContext.formatLog(fmt.Sprintf("error occurred extracting bala file: %s", err.Error())))
	}

	if err := bfs.Remove(fsys, balaPath); err != nil {
		return NewCentralClientError(clientContext.formatLog(fmt.Sprintf("error occurred extracting bala file: %s", err.Error())))
	}

	return nil
}

func writeAndHandleProgress(inputStream io.Reader, totalSizeInBytes int64, fsys fs.FS, balaPath string, clientContext ClientContext,
) error {
	if totalSizeInBytes <= 0 {
		return fmt.Errorf("invalid content length received from the server")
	}

	file := make([]byte, 0, totalSizeInBytes)
	buffer := make([]byte, 1024)

	var totalRead int64

	for {
		n, err := inputStream.Read(buffer)

		if n > 0 {
			file = append(file, buffer[:n]...)
			totalRead = totalRead + int64(n)

			progress := int((totalRead * 100) / totalSizeInBytes)
			if clientContext.OnProgress != nil {
				clientContext.OnProgress(progress)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading stream: %w", err)
		}
	}

	if err := bfs.WriteFile(fsys, balaPath, file, 0o644); err != nil {
		return fmt.Errorf("error occurred copying bala file: %w", err)
	}

	return nil
}

func extractBala(fsys fs.FS, balaFilePath, balaFileDestPath, trueDigest, packageName string, clientContext ClientContext) error {
	if err := bfs.MkdirAll(fsys, balaFileDestPath, 0o755); err != nil {
		return err
	}

	hash, err := checkHashInternal(fsys, balaFilePath)
	if err != nil {
		return err
	}

	actualDigest := fmt.Sprintf("%s%s", SHA256, hash)
	if trueDigest != "" && trueDigest != actualDigest {
		if clientContext.OnWarning != nil {
			warning := fmt.Sprintf(`*************************************************************
* WARNING: Certain packages may have originated from sources other than the official distributors. *
*************************************************************

* Verification failed: The hash value of the following package could not be confirmed. 
%s
`, packageName)
			clientContext.OnWarning(warning)
		}
	}

	file, err := fs.ReadFile(fsys, balaFilePath)
	if err != nil {
		return err
	}

	reader, err := zip.NewReader(bytes.NewReader(file), int64(len(file)))
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		filePath := path.Join(balaFileDestPath, file.Name)

		if file.FileInfo().IsDir() {
			_ = bfs.MkdirAll(fsys, filePath, file.Mode())
			continue
		}

		if err := bfs.MkdirAll(fsys, path.Dir(filePath), 0o755); err != nil {
			return err
		}

		outFile, err := bfs.OpenFile(fsys, filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		rc, err := file.Open()
		if err != nil {
			_ = outFile.Close()
			return err
		}

		err = bfs.WriteFile(fsys, filePath, func() []byte {
			data, _ := io.ReadAll(rc)
			return data
		}(), file.Mode())
		_ = outFile.Close()
		_ = rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func handleNightlyBuild(isNightlyBuild bool, fsys fs.FS, balaCacheWithPkgPath string, clientContext ClientContext) error {
	if isNightlyBuild {
		nightlyBuildMetaFile := path.Join(balaCacheWithPkgPath, "nightly.build")
		if _, err := fs.Stat(fsys, nightlyBuildMetaFile); os.IsNotExist(err) {
			errMsg := "error occurred while creating nightly.build file."
			return createMetaFile(fsys, nightlyBuildMetaFile, errMsg, clientContext)
		}
	}
	return nil
}

func handlePackageDeprecation(deprecateMsg string, fsys fs.FS, balaCacheWithPkgPath string, clientContext ClientContext) error {
	if deprecateMsg != "" {
		deprecateMsgFile := path.Join(balaCacheWithPkgPath, DeprecatedMetaFileName)
		if _, err := fs.Stat(fsys, deprecateMsgFile); os.IsNotExist(err) {
			errMsg := fmt.Sprintf("error occurred while creating the file '%s'.", DeprecatedMetaFileName)
			if err := createMetaFile(fsys, deprecateMsgFile, errMsg, clientContext); err != nil {
				return err
			}
		}
		return writeDeprecatedMsg(fsys, deprecateMsgFile, deprecateMsg, clientContext)
	}
	return nil
}

func writeDeprecatedMsg(fsys fs.FS, metaFilePath string, message string, clientContext ClientContext) error {
	if _, err := fs.Stat(fsys, metaFilePath); err == nil {
		if err := bfs.WriteFile(fsys, metaFilePath, []byte(message), 0o644); err != nil {
			return NewCentralClientError(
				clientContext.formatLog(fmt.Sprintf("error occurred while writing deprecation message to the file '%s': %s", DeprecatedMetaFileName, err.Error())))
		}
	}
	return nil
}

func createMetaFile(fsys fs.FS, metaFilePath string, errMsg string, clientContext ClientContext) error {
	file, err := bfs.Create(fsys, metaFilePath)
	if err != nil {
		return NewCentralClientError(clientContext.formatLog(errMsg))
	}
	defer func() { _ = file.Close() }()
	return nil
}

func checkHashInternal(fsys fs.FS, filePath string) (string, error) {
	file, err := fsys.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func getAsList(arrayString string) ([]string, error) {
	var list []string
	if err := json.Unmarshal([]byte(arrayString), &list); err != nil {
		return nil, err
	}
	return list, nil
}
