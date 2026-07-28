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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"ballerina/projects/centralclient/models"
)

type CentralAPIClient interface {
	GetPackage(orgNamePath, packageNamePath, version, supportedPlatform, ballerinaVersion string) (*models.Package, error)
	GetPackageVersions(orgNamePath, packageNamePath, supportedPlatform, ballerinaVersion string) ([]string, error)
	PullPackage(org, name, version string, fsys fs.FS, packagePathInBalaCache, supportedPlatform, ballerinaVersion string, clientContext ClientContext) error
	ResolvePackageNames(request models.PackageNameResolutionRequest, supportedPlatform, ballerinaVersion string) (*models.PackageNameResolutionResponse, error)
	ResolveDependencies(request models.PackageResolutionRequest, supportedPlatform, ballerinaVersion string) (*models.PackageResolutionResponse, error)
	GetConnectors(params map[string]string, supportedPlatform, ballerinaVersion string) (any, error)
	GetConnector(id, supportedPlatform, ballerinaVersion string) (map[string]any, error)
	GetConnectorByInfo(connector models.ConnectorInfo, supportedPlatform, ballerinaVersion string) (map[string]any, error)
	GetTriggers(params map[string]string, supportedPlatform, ballerinaVersion string) (any, error)
	GetTrigger(id, supportedPlatform, ballerinaVersion string) (map[string]any, error)
	AccessToken() string
	SetAccessToken(token string)
}

type centralAPIClientImpl struct {
	baseURL        string
	proxyURL       string
	accessToken    string
	proxyUsername  string
	proxyPassword  string
	connectTimeout time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
	callTimeout    time.Duration
	maxRetries     int
	httpClient     http.Client
}

type ClientContext struct {
	OnRetry    func(retryCount int)
	OnProgress func(percentComplete int)
	OnWarning  func(msg string)
	IsBuild    bool
}

func (c *ClientContext) formatLog(message string) string {
	if c.IsBuild {
		return fmt.Sprintf("\t%s", message)
	}
	return message
}

func NewCentralAPIClient(baseURL string, proxyURL string, accessToken string) CentralAPIClient {
	httpClient := buildHTTPClient(baseURL, proxyURL, "", "", DefaultCallTimeout*time.Second, MaxRetry)
	return &centralAPIClientImpl{
		baseURL:        baseURL,
		proxyURL:       proxyURL,
		accessToken:    accessToken,
		connectTimeout: DefaultConnectTimeout * time.Second,
		readTimeout:    DefaultReadTimeout * time.Second,
		writeTimeout:   DefaultWriteTimeout * time.Second,
		callTimeout:    DefaultCallTimeout * time.Second,
		maxRetries:     MaxRetry,
		httpClient:     httpClient,
	}
}

func NewCentralAPIClientFull(baseURL string, proxyURL string, proxyUsername, proxyPassword, accessToken string, connectTimeout, readTimeout, writeTimeout, callTimeout, maxRetries int) CentralAPIClient {
	httpClient := buildHTTPClient(baseURL, proxyURL, proxyUsername, proxyPassword, time.Duration(callTimeout)*time.Second, maxRetries)
	return &centralAPIClientImpl{
		baseURL:        baseURL,
		proxyURL:       proxyURL,
		accessToken:    accessToken,
		proxyUsername:  proxyUsername,
		proxyPassword:  proxyPassword,
		connectTimeout: time.Duration(connectTimeout) * time.Second,
		readTimeout:    time.Duration(readTimeout) * time.Second,
		writeTimeout:   time.Duration(writeTimeout) * time.Second,
		callTimeout:    time.Duration(callTimeout) * time.Second,
		maxRetries:     maxRetries,
		httpClient:     httpClient,
	}
}

func (c *centralAPIClientImpl) GetPackage(orgNamePath, packageNamePath, version, supportedPlatform, ballerinaVersion string) (*models.Package, error) {
	pkg, err := c.getPackageInternal(orgNamePath, packageNamePath, version, supportedPlatform, ballerinaVersion)
	if err != nil {
		switch err.(type) {
		case *NoPackageError, *CentralClientError:
			return nil, err

		default:
			return nil, NewCentralClientError(fmt.Sprintf("%s%s", ErrCannotFindPackage, getPackageSignature(orgNamePath, packageNamePath, version)))
		}
	}

	return pkg, nil
}

func (c *centralAPIClientImpl) getPackageInternal(orgNamePath, packageNamePath, version, supportedPlatform, ballerinaVersion string) (*models.Package, error) {
	resourceURL := fmt.Sprintf("%s%s%s%s", PackagePathPrefix, orgNamePath, Separator, packageNamePath)

	urlStr := fmt.Sprintf("%s%s", c.baseURL, resourceURL)
	if version != "" {
		urlStr = fmt.Sprintf("%s/%s", urlStr, version)
	}

	req, err := c.newRequest(http.MethodGet, urlStr, supportedPlatform, ballerinaVersion, nil)
	if err != nil {
		return nil, err
	}

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, resourceURL)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logResponseVerbose(resp, string(bodyBytes))

	contentType := resp.Header.Get(ContentType)
	if isApplicationJSONContentType(contentType) {
		switch resp.StatusCode {
		case http.StatusOK:
			var pkg *models.Package
			if err := json.Unmarshal(bodyBytes, &pkg); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s%s. reason: unexpected error", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, version)))
			}
			return pkg, nil

		case http.StatusNotFound:
			var errResp models.Error
			if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s%s. reason: unexpected error", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, version)))
			}

			if errResp.Message != "" && strings.Contains(errResp.Message, "package not found for:") {
				return nil, NewNoPackageError(errResp.Message)
			}

		case http.StatusUnauthorized:
			return nil, c.handleUnauthorizedResponseWithOrg(orgNamePath, bodyBytes)

		case http.StatusBadRequest, http.StatusInternalServerError, http.StatusServiceUnavailable:
			var errResp models.Error

			if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s%s. reason: unexpected error", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, version)))
			}

			if errResp.Message != "" {
				return nil, NewCentralClientError(fmt.Sprintf("%s%s. reason: %s", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, version), errResp.Message))
			}
		}
	}

	return nil, NewCentralClientError(fmt.Sprintf("%s%s", ErrCannotFindPackage, getPackageSignature(orgNamePath, packageNamePath, version)))
}

func (c *centralAPIClientImpl) GetPackageVersions(orgNamePath, packageNamePath, supportedPlatform, ballerinaVersion string) ([]string, error) {
	versions, err := c.getPackageVersionsInternal(orgNamePath, packageNamePath, supportedPlatform, ballerinaVersion)
	if err != nil {
		return nil, wrapCentralClientError(err, fmt.Sprintf("%s%s", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, "")))
	}

	return versions, nil
}

func (c *centralAPIClientImpl) getPackageVersionsInternal(orgNamePath, packageNamePath, supportedPlatform, ballerinaVersion string) ([]string, error) {
	resourceURL := fmt.Sprintf("%s%s%s%s", PackagePathPrefix, orgNamePath, Separator, packageNamePath)

	urlStr := fmt.Sprintf("%s%s", c.baseURL, resourceURL)

	req, err := c.newRequest(http.MethodGet, urlStr, supportedPlatform, ballerinaVersion, nil)
	if err != nil {
		return nil, err
	}

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, resourceURL)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logResponseVerbose(resp, string(bodyBytes))

	contentType := resp.Header.Get(ContentType)
	if isApplicationJSONContentType(contentType) {
		switch resp.StatusCode {
		case http.StatusOK:
			var versions []string
			if err := json.Unmarshal(bodyBytes, &versions); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s%s. reason: unexpected error", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, "")))
			}
			return versions, nil

		case http.StatusUnauthorized:
			return nil, c.handleUnauthorizedResponseWithOrg(orgNamePath, bodyBytes)

		case http.StatusNotFound:
			var errResp models.Error

			if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s%s. reason: unexpected error", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, "")))
			}

			if !strings.Contains(errResp.Message, "package not found:") {
				return nil, NewCentralClientError(fmt.Sprintf("%s%s. reason: %s", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, ""), errResp.Message))
			}

			return nil, nil

		case http.StatusBadRequest, http.StatusInternalServerError, http.StatusServiceUnavailable:
			var errResp models.Error

			if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s%s. reason: unexpected error", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, "")))
			}

			if errResp.Message != "" {
				return nil, NewCentralClientError(fmt.Sprintf("%s%s. reason: %s", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, ""), errResp.Message))
			}
		}
	}

	return nil, NewCentralClientError(fmt.Sprintf("%s%s", ErrCannotFindVersions, getPackageSignature(orgNamePath, packageNamePath, "")))
}

func (c *centralAPIClientImpl) PullPackage(org, name, version string, fsys fs.FS, packagePathInBalaCache, supportedPlatform, ballerinaVersion string, clientContext ClientContext) error {
	for retryCount := 0; retryCount <= c.maxRetries; retryCount++ {
		err := c.pullPackageInternal(org, name, version, fsys, packagePathInBalaCache, supportedPlatform, ballerinaVersion, clientContext)

		if err == nil {
			return nil
		}

		if !strings.Contains(err.Error(), ConnectionReset) {
			return err
		}

		if retryCount >= c.maxRetries {
			return err
		}

		if clientContext.OnRetry != nil {
			clientContext.OnRetry(retryCount + 1)
		}
	}

	return nil
}

func (c *centralAPIClientImpl) ResolvePackageNames(request models.PackageNameResolutionRequest, supportedPlatform, ballerinaVersion string) (*models.PackageNameResolutionResponse, error) {
	response, err := c.resolvePackageNamesInternal(request, supportedPlatform, ballerinaVersion)
	if err != nil {
		return nil, wrapCentralOrConnectionError(err, ErrPackageResolution)
	}

	return response, nil
}

func (c *centralAPIClientImpl) resolvePackageNamesInternal(request models.PackageNameResolutionRequest, supportedPlatform, ballerinaVersion string) (*models.PackageNameResolutionResponse, error) {
	urlStr := fmt.Sprintf("%s%s%s", c.baseURL, PackagePathPrefix, ResolveModules)

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return nil, NewCentralClientError(fmt.Sprintf("%s%s", ErrPackageResolution, err.Error()))
	}

	req, err := c.newRequest(http.MethodPost, urlStr, supportedPlatform, ballerinaVersion, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, NewCentralClientError(fmt.Sprintf("%s%s", ErrPackageResolution, err.Error()))
	}

	req.Header.Set(ContentType, MediaTypeJSON)
	req.Header.Set(Accept, MediaTypeJSONContent)
	req.Header.Set(AcceptEncoding, Identity)

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, NewCentralClientError(fmt.Sprintf("%s%s", ErrPackageResolution, err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, fmt.Sprintf("%s%s", PackagePathPrefix, ResolveModules))

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewCentralClientError(fmt.Sprintf("%s%s", ErrPackageResolution, err.Error()))
	}

	c.logResponseVerbose(resp, string(respBodyBytes))

	contentType := resp.Header.Get(ContentType)
	if isApplicationJSONContentType(contentType) {
		switch resp.StatusCode {
		case http.StatusOK:
			var resolutionResponse *models.PackageNameResolutionResponse
			if err := json.Unmarshal(respBodyBytes, &resolutionResponse); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrPackageResolution))
			}
			return resolutionResponse, nil

		case http.StatusUnauthorized:
			return nil, c.handleUnauthorizedResponse(bodyBytes)

		case http.StatusBadRequest:
			var errResp models.Error
			if err := json.Unmarshal(respBodyBytes, &errResp); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrPackageResolution))
			}

			if errResp.Message != "" {
				return nil, NewConnectionError(errResp.Message)
			}

		case http.StatusInternalServerError, http.StatusServiceUnavailable:
			var errResp models.Error
			if err := json.Unmarshal(respBodyBytes, &errResp); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrPackageResolution))
			}

			if errResp.Message != "" {
				return nil, NewConnectionError(fmt.Sprintf("%s. reason: %s", ErrPackageResolution, errResp.Message))
			}
		}
	}

	return nil, NewConnectionError(ErrPackageResolution)
}

func (c *centralAPIClientImpl) ResolveDependencies(request models.PackageResolutionRequest, supportedPlatform, ballerinaVersion string) (*models.PackageResolutionResponse, error) {
	response, err := c.resolveDependenciesInternal(request, supportedPlatform, ballerinaVersion)
	if err != nil {
		return nil, wrapCentralOrConnectionError(err, ErrPackageResolution)
	}

	return response, nil
}

func (c *centralAPIClientImpl) resolveDependenciesInternal(request models.PackageResolutionRequest, supportedPlatform, ballerinaVersion string) (*models.PackageResolutionResponse, error) {
	urlStr := fmt.Sprintf("%s%s%s", c.baseURL, PackagePathPrefix, ResolveDependencies)

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(http.MethodPost, urlStr, supportedPlatform, ballerinaVersion, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set(AcceptEncoding, Identity)
	req.Header.Set(Accept, ApplicationJSON)

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, fmt.Sprintf("%s%s", PackagePathPrefix, ResolveDependencies))

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logResponseVerbose(resp, string(respBodyBytes))

	contentType := resp.Header.Get(ContentType)
	if isApplicationJSONContentType(contentType) {
		switch resp.StatusCode {
		case http.StatusOK:
			var resolutionResponse *models.PackageResolutionResponse
			if err := json.Unmarshal(respBodyBytes, &resolutionResponse); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrPackageResolution))
			}
			return resolutionResponse, nil

		case http.StatusUnauthorized:
			return nil, c.handleUnauthorizedResponse(bodyBytes)

		case http.StatusBadRequest:
			var errResp models.Error
			if err := json.Unmarshal(respBodyBytes, &errResp); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrPackageResolution))
			}

			if errResp.Message != "" {
				return nil, NewConnectionError(errResp.Message)
			}

		case http.StatusInternalServerError, http.StatusServiceUnavailable:
			var errResp models.Error
			if err := json.Unmarshal(respBodyBytes, &errResp); err != nil {
				return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrPackageResolution))
			}

			if errResp.Message != "" {
				return nil, NewConnectionError(fmt.Sprintf("%s. reason: %s", ErrPackageResolution, errResp.Message))
			}
		}
	}

	return nil, NewConnectionError(ErrPackageResolution)
}

func (c *centralAPIClientImpl) GetConnectors(params map[string]string, supportedPlatform, ballerinaVersion string) (any, error) {
	connectors, err := c.getConnectorsInternal(params, supportedPlatform, ballerinaVersion)
	if err != nil {
		return nil, wrapCentralClientError(err, ErrCannotGetConnector)
	}

	return connectors, nil
}

func (c *centralAPIClientImpl) getConnectorsInternal(params map[string]string, supportedPlatform, ballerinaVersion string) (any, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(baseURL.Path, ConnectorsPath)
	query := baseURL.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	baseURL.RawQuery = query.Encode()

	req, err := c.newRequest(http.MethodGet, baseURL.String(), supportedPlatform, ballerinaVersion, nil)
	if err != nil {
		return nil, err
	}

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, fmt.Sprintf("%s%s", Separator, ConnectorsPath))

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logResponseVerbose(resp, string(bodyBytes))

	contentType := resp.Header.Get(ContentType)
	if isApplicationJSONContentType(contentType) && resp.StatusCode == http.StatusOK {
		var connectors any
		if err := json.Unmarshal(bodyBytes, &connectors); err != nil {
			return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrCannotGetConnector))
		}
		return connectors, nil
	}

	return nil, c.handleResponseErrors(resp, ErrCannotGetConnector, bodyBytes)
}

func (c *centralAPIClientImpl) GetConnector(id, supportedPlatform, ballerinaVersion string) (map[string]any, error) {
	connector, err := c.getConnectorInternal(id, supportedPlatform, ballerinaVersion)
	if err != nil {
		return nil, wrapCentralClientError(err, fmt.Sprintf("%sid: %s", ErrCannotGetConnector, id))
	}

	return connector, nil
}

func (c *centralAPIClientImpl) getConnectorInternal(id, supportedPlatform, ballerinaVersion string) (map[string]any, error) {
	urlStr := fmt.Sprintf("%s%s%s", c.baseURL, ConnectorPathPrefix, id)

	req, err := c.newRequest(http.MethodGet, urlStr, supportedPlatform, ballerinaVersion, nil)
	if err != nil {
		return nil, err
	}

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, fmt.Sprintf("%s%s", ConnectorPathPrefix, id))

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logResponseVerbose(resp, string(bodyBytes))

	contentType := resp.Header.Get(ContentType)

	if isApplicationJSONContentType(contentType) && resp.StatusCode == http.StatusOK {
		var connector map[string]any
		if err := json.Unmarshal(bodyBytes, &connector); err != nil {
			return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrCannotGetConnector))
		}
		return connector, nil
	}

	return nil, c.handleResponseErrors(resp, fmt.Sprintf("%sid: %s", ErrCannotGetConnector, id), bodyBytes)
}

func (c *centralAPIClientImpl) GetConnectorByInfo(connector models.ConnectorInfo, supportedPlatform, ballerinaVersion string) (map[string]any, error) {
	result, err := c.getConnectorByInfoInternal(connector, supportedPlatform, ballerinaVersion)
	if err != nil {
		return nil, wrapCentralClientError(err, fmt.Sprintf("%s'%s'", ErrCannotGetConnector, connector.PackageName))
	}

	return result, nil
}

func (c *centralAPIClientImpl) getConnectorByInfoInternal(connector models.ConnectorInfo, supportedPlatform, ballerinaVersion string) (map[string]any, error) {
	resourcePath := ConnectorPathPrefix + connector.OrgName + Separator + connector.PackageName + Separator + connector.Version + Separator + connector.ModuleName + Separator + connector.Name
	urlStr := fmt.Sprintf("%s%s", c.baseURL, resourcePath)

	req, err := c.newRequest(http.MethodGet, urlStr, supportedPlatform, ballerinaVersion, nil)
	if err != nil {
		return nil, err
	}

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, resourcePath)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logResponseVerbose(resp, string(bodyBytes))

	contentType := resp.Header.Get(ContentType)
	if isApplicationJSONContentType(contentType) && resp.StatusCode == http.StatusOK {
		var connector map[string]any
		if err := json.Unmarshal(bodyBytes, &connector); err != nil {
			return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrCannotGetConnector))
		}
		return connector, nil
	}

	return nil, c.handleResponseErrors(resp, fmt.Sprintf("%s'%s'", ErrCannotGetConnector, connector.PackageName), bodyBytes)
}

func (c *centralAPIClientImpl) GetTriggers(params map[string]string, supportedPlatform, ballerinaVersion string) (any, error) {
	triggers, err := c.getTriggersInternal(params, supportedPlatform, ballerinaVersion)
	if err != nil {
		return nil, wrapCentralClientError(err, ErrCannotGetTriggers)
	}

	return triggers, nil
}

func (c *centralAPIClientImpl) getTriggersInternal(params map[string]string, supportedPlatform, ballerinaVersion string) (any, error) {
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}

	baseURL.Path = path.Join(baseURL.Path, TriggersPath)
	query := baseURL.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	baseURL.RawQuery = query.Encode()

	req, err := c.newRequest(http.MethodGet, baseURL.String(), supportedPlatform, ballerinaVersion, nil)
	if err != nil {
		return nil, err
	}

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, fmt.Sprintf("%s%s", Separator, ConnectorsPath))

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logResponseVerbose(resp, string(bodyBytes))

	contentType := resp.Header.Get(ContentType)
	if isApplicationJSONContentType(contentType) && resp.StatusCode == http.StatusOK {
		var connectors any
		if err := json.Unmarshal(bodyBytes, &connectors); err != nil {
			return nil, NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", ErrCannotGetTriggers))
		}
		return connectors, nil
	}

	return nil, c.handleResponseErrors(resp, ErrCannotGetTriggers, bodyBytes)
}

func (c *centralAPIClientImpl) GetTrigger(id, supportedPlatform, ballerinaVersion string) (map[string]any, error) {
	trigger, err := c.getTriggerInternal(id, supportedPlatform, ballerinaVersion)
	if err != nil {
		return nil, wrapCentralClientError(err, fmt.Sprintf("%s id: %s", ErrCannotGetTrigger, id))
	}

	return trigger, nil
}

func (c *centralAPIClientImpl) getTriggerInternal(id, supportedPlatform, ballerinaVersion string) (map[string]any, error) {
	urlStr := fmt.Sprintf("%s%s%s", c.baseURL, TriggerPathPrefix, id)

	req, err := c.newRequest(http.MethodGet, urlStr, supportedPlatform, ballerinaVersion, nil)
	if err != nil {
		return nil, err
	}

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, fmt.Sprintf("%s%s", TriggerPathPrefix, id))

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	c.logResponseVerbose(resp, string(bodyBytes))

	contentType := resp.Header.Get(ContentType)
	if isApplicationJSONContentType(contentType) && resp.StatusCode == http.StatusOK {
		var trigger map[string]any
		if err := json.Unmarshal(bodyBytes, &trigger); err != nil {
			return nil, NewCentralClientError(fmt.Sprintf("%s reason: unexpected error", ErrCannotGetTrigger))
		}
		return trigger, nil
	}

	return nil, c.handleResponseErrors(resp, fmt.Sprintf("%sid: %s", ErrCannotGetTrigger, id), bodyBytes)
}

func (c *centralAPIClientImpl) AccessToken() string {
	return c.accessToken
}

func (c *centralAPIClientImpl) SetAccessToken(token string) {
	c.accessToken = token
}

// wrapCentralClientError wraps an error with CentralClientError unless it's already a CentralClientError.
func wrapCentralClientError(err error, message string) error {
	if _, ok := err.(*CentralClientError); ok {
		return err
	}
	return NewCentralClientError(message)
}

// wrapCentralOrConnectionError wraps an error with CentralClientError unless it's already
// a CentralClientError or ConnectionError.
func wrapCentralOrConnectionError(err error, message string) error {
	switch err.(type) {
	case *CentralClientError, *ConnectionError:
		return err
	default:
		return NewCentralClientError(message)
	}
}

func (c *centralAPIClientImpl) pullPackageInternal(org, name, version string, fsys fs.FS, packagePathInBalaCache, supportedPlatform, ballerinaVersion string, clientContext ClientContext) error {
	resourceURL := fmt.Sprintf("%s%s%s%s", PackagePathPrefix, org, Separator, name)

	urlStr := fmt.Sprintf("%s%s", c.baseURL, resourceURL)

	if version != "" {
		urlStr = fmt.Sprintf("%s/%s", urlStr, version)
	} else {
		urlStr = fmt.Sprintf("%s/%s", urlStr, "*")
	}

	req, err := c.newRequest(http.MethodGet, urlStr, supportedPlatform, ballerinaVersion, nil)
	if err != nil {
		return NewCentralClientError(clientContext.formatLog(fmt.Sprintf("%s'%s'", ErrCannotPullPackage, getPackageSignature(org, name, version))))
	}

	req.Header.Set(AcceptEncoding, Identity)
	req.Header.Set(Accept, ApplicationOctetStream)

	c.logRequestInitVerbose(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return NewCentralClientError(clientContext.formatLog(fmt.Sprintf("%s'%s'", ErrCannotPullPackage, getPackageSignature(org, name, version))))
	}
	defer func() { _ = resp.Body.Close() }()

	c.logRequestConnectVerbose(req, resourceURL)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewCentralClientError(clientContext.formatLog(fmt.Sprintf("%s'%s'", ErrCannotPullPackage, getPackageSignature(org, name, version))))
	}

	c.logResponseVerbose(resp, string(bodyBytes))

	if resp.StatusCode == http.StatusFound {
		balaURL := resp.Header.Get(Location)
		balaFileName := resp.Header.Get(ContentDisposition)
		deprecationFlag := resp.Header.Get("Is-Deprecated")
		deprecationMsg := resp.Header.Get("Deprecate-Message")
		digest := resp.Header.Get(Digest)

		isDeprecated := deprecationFlag == "true"
		deprecationMessage := deprecationMsg

		if !clientContext.IsBuild && isDeprecated {
			if clientContext.OnWarning != nil {
				clientContext.OnWarning(fmt.Sprintf("WARNING [%s] %s is deprecated: %s",
					name, getPackageSignature(org, name, version), deprecationMessage))
			}
		}

		if balaURL != "" && balaFileName != "" {
			downloadReq, err := c.newRequest(http.MethodGet, balaURL, supportedPlatform, ballerinaVersion, nil)
			if err != nil {
				return NewCentralClientError(clientContext.formatLog(fmt.Sprintf("%s'%s'", ErrCannotPullPackage, getPackageSignature(org, name, version))))
			}

			downloadReq.Header.Set(AcceptEncoding, Identity)
			downloadReq.Header.Set(ContentDisposition, balaFileName)

			c.logRequestInitVerbose(downloadReq)

			downloadResp, err := c.httpClient.Do(downloadReq)
			if err != nil {
				return NewCentralClientError(clientContext.formatLog(fmt.Sprintf("%s'%s': %s", ErrCannotPullPackage, getPackageSignature(org, name, version), err.Error())))
			}
			defer func() { _ = downloadResp.Body.Close() }()

			c.logRequestConnectVerbose(downloadReq, balaURL)
			c.logResponseVerbose(downloadResp, "")

			if downloadResp.StatusCode == http.StatusOK {
				isNightlyBuild := strings.Contains(ballerinaVersion, "SNAPSHOT")

				deprecMsg := ""
				if isDeprecated {
					deprecMsg = deprecationMessage
				}

				return createBalaInHomeRepo(downloadResp, fsys, packagePathInBalaCache, org, name, isNightlyBuild,
					deprecMsg, balaURL, balaFileName, digest, clientContext)
			}

			errorMsg := clientContext.formatLog(fmt.Sprintf("%s'%s'. BALA content download from '%s' failed.", ErrCannotPullPackage, getPackageSignature(org, name, version), balaURL))
			return c.handleResponseErrors(downloadResp, errorMsg, bodyBytes)
		}

		errorMsg := clientContext.formatLog(fmt.Sprintf("%s'%s'. BALA content download from '%s' failed.", ErrCannotPullPackage, getPackageSignature(org, name, version), balaURL))
		return NewCentralClientError(errorMsg)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return c.handleUnauthorizedResponseWithOrg(org, bodyBytes)
	}

	contentType := resp.Header.Get(ContentType)
	if isApplicationJSONContentType(contentType) {
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound {
			var errResp models.Error
			if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Message != "" {
				return NewCentralClientError(fmt.Sprintf("error: %s", errResp.Message))
			}
		}

		if resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == http.StatusServiceUnavailable {
			var errResp models.Error
			if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Message != "" {
				errorMsg := clientContext.formatLog(fmt.Sprintf("%s'%s' from the remote repository '%s'. reason: %s",
					ErrCannotPullPackage, getPackageSignature(org, name, version), urlStr, errResp.Message))
				return NewCentralClientError(errorMsg)
			}
		}
	}

	errorMsg := clientContext.formatLog(fmt.Sprintf("%s'%s' from the remote repository '%s'.",
		ErrCannotPullPackage, getPackageSignature(org, name, version), urlStr))
	return NewCentralClientError(errorMsg)
}

func buildHTTPClient(baseURL, proxyURL, proxyUsername, proxyPassword string, callTimeout time.Duration, maxRetries int) http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}

	if proxyURL != "" {
		parsedProxyURL, err := url.Parse(proxyURL)
		if err == nil {
			if proxyUsername != "" && proxyPassword != "" {
				parsedProxyURL.User = url.UserPassword(proxyUsername, proxyPassword)
			}
			transport.Proxy = http.ProxyURL(parsedProxyURL)
		}
	}

	client := http.Client{
		Transport: &customRetryTransport{
			transport:  transport,
			maxRetries: maxRetries,
			baseURL:    baseURL,
		},
		Timeout: callTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return client
}

type customRetryTransport struct {
	transport  http.RoundTripper
	maxRetries int
	baseURL    string
}

func (r *customRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	retryCount := 0

	for retryCount <= r.maxRetries {
		// Clone request body for potential retries
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, err = io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			_ = req.Body.Close()
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		resp, err = r.transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode < 500 || retryCount == r.maxRetries {
			return resp, nil
		}

		var bodyContent string
		if resp.Body != nil {
			bodyBytes, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr == nil {
				bodyContent = string(bodyBytes)
			}
		}

		r.logRetryVerbose(resp, bodyContent, req, retryCount+1)

		retryCount = retryCount + 1

		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	return resp, err
}

func (c *centralAPIClientImpl) newRequest(method, urlStr, supportedPlatform, ballerinaVersion string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set(BallerinaPlatform, supportedPlatform)
	req.Header.Set(UserAgent, ballerinaVersion)
	req.Header.Set(BallerinaCentralTelemetryDisabled, strconv.FormatBool(os.Getenv(TestModeActive) == "true"))

	if c.accessToken != "" {
		req.Header.Set(Authorization, getBearerToken(c.accessToken))
	}

	return req, nil
}

func (c *centralAPIClientImpl) handleResponseErrors(resp *http.Response, msg string, bodyBytes []byte) error {
	contentType := resp.Header.Get(ContentType)

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound {
		if isApplicationJSONContentType(contentType) {
			var errResp models.Error
			if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
				return NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", msg))
			}

			if errResp.Message != "" {
				return NewCentralClientError(fmt.Sprintf("%s. reason: %s", msg, errResp.Message))
			}
		}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if isApplicationJSONContentType(contentType) {
			return c.handleUnauthorizedResponse(bodyBytes)
		}
		return NewCentralClientError("unauthorized access token. check access token set in 'Settings.toml' file.")
	}

	if resp.StatusCode == http.StatusInternalServerError || resp.StatusCode == http.StatusServiceUnavailable ||
		resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusGatewayTimeout {
		if isApplicationJSONContentType(contentType) {
			var errResp models.Error
			if err := json.Unmarshal(bodyBytes, &errResp); err != nil {
				return NewCentralClientError(fmt.Sprintf("%s. reason: unexpected error", msg))
			}

			if errResp.Message != "" {
				return NewCentralClientError(fmt.Sprintf("%s. reason: %s", msg, errResp.Message))
			}
		}
	}

	return NewCentralClientError(msg)
}

func (c *centralAPIClientImpl) handleUnauthorizedResponse(bodyBytes []byte) error {
	var errResp models.Error
	if err := json.Unmarshal(bodyBytes, &errResp); err == nil {
		return NewCentralClientError(fmt.Sprintf("unauthorized access token. check access token set in 'Settings.toml' file. reason: %s", errResp.Message))
	}
	return NewCentralClientError("unauthorized access token. check access token set in 'Settings.toml' file.")
}

func (c *centralAPIClientImpl) handleUnauthorizedResponseWithOrg(org string, bodyBytes []byte) error {
	var errResp models.Error
	if err := json.Unmarshal(bodyBytes, &errResp); err == nil {
		return NewCentralClientError(fmt.Sprintf("unauthorized access token for organization: '%s'. check access token set in 'Settings.toml' file. reason: %s", org, errResp.Message))
	}
	return NewCentralClientError(fmt.Sprintf("unauthorized access token for organization: '%s'. check access token set in 'Settings.toml' file.", org))
}

func (r *customRetryTransport) logRetryVerbose(resp *http.Response, bodyContent string, req *http.Request, retryCount int) {
	if !isVerboseEnabled() {
		return
	}

	fmt.Fprintf(os.Stderr, "< HTTP %d %s\n", resp.StatusCode, resp.Status)

	for name, values := range resp.Header {
		for _, value := range values {
			fmt.Fprintf(os.Stderr, "> %s: %s\n", name, value)
		}
	}

	fmt.Fprintln(os.Stderr, "< ")

	if bodyContent != "" {
		fmt.Fprintln(os.Stderr, bodyContent)
	}

	fmt.Fprintf(os.Stderr, "* Connection to host %s left intact \n\n", r.baseURL)
	fmt.Fprintf(os.Stderr, "* Retrying request to %s due to %d response code. Retry attempt: %d\n",
		req.URL.String(), resp.StatusCode, retryCount)
}

func (c *centralAPIClientImpl) logRequestInitVerbose(req *http.Request) {
	if isVerboseEnabled() {
		fmt.Fprintf(os.Stderr, "* Trying %s\n", req.URL.String())
	}
}

func (c *centralAPIClientImpl) logRequestConnectVerbose(req *http.Request, resourceURL string) {
	if isVerboseEnabled() {
		fmt.Fprintf(os.Stderr, "* Connected to %s\n", c.baseURL)
		fmt.Fprintf(os.Stderr, "> %s %s HTTP\n", req.Method, resourceURL)
		fmt.Fprintf(os.Stderr, "> Host: %s\n", c.baseURL)
		for name, values := range req.Header {
			for _, value := range values {
				if name == "Authorization" {
					fmt.Fprintf(os.Stderr, "> %s: Bearer ************************************\n", name)
				} else {
					fmt.Fprintf(os.Stderr, "> %s: %s\n", name, value)
				}
			}
		}
		fmt.Fprintln(os.Stderr, ">")
	}
}

func (c *centralAPIClientImpl) logResponseVerbose(resp *http.Response, bodyContent string) {
	if isVerboseEnabled() {
		fmt.Fprintf(os.Stderr, "< HTTP %s\n", resp.Status)

		for name, values := range resp.Header {
			for _, value := range values {
				fmt.Fprintf(os.Stderr, "> %s: %s\n", name, value)
			}
		}
		fmt.Fprintln(os.Stderr, "< ")
		if bodyContent != "" {
			fmt.Fprintln(os.Stderr, bodyContent)
		}
		fmt.Fprintf(os.Stderr, "* Connection to host %s left intact\n\n", c.baseURL)
	}
}

func isVerboseEnabled() bool {
	return os.Getenv("CENTRAL_VERBOSE_ENABLED") == "true"
}

func getBearerToken(accessToken string) string {
	return fmt.Sprintf("Bearer %s", accessToken)
}

func isApplicationJSONContentType(contentType string) bool {
	return strings.HasPrefix(contentType, MediaTypeJSONContent)
}

func getPackageSignature(org, name, version string) string {
	if version != "" {
		return fmt.Sprintf("%s%s%s:%s", org, Separator, name, version)
	}
	return fmt.Sprintf("%s%s%s", org, Separator, name)
}
