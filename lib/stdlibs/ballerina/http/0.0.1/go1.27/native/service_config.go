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

package native

import (
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/model"
	"github.com/ballerina-nutcracker/ballerina/runtime/extern"
	"github.com/ballerina-nutcracker/ballerina/values"
)

var serviceConfigAnnotation = model.AnnotationKey(httpPackage, "ServiceConfig")

// defaultHost is the host of a service that names none; it serves requests whose Host header
// matches no other service.
const defaultHost = "b7a.default"

// serviceConfig holds the @http:ServiceConfig fields the dispatcher honours.
type serviceConfig struct {
	// host is lower-cased, without brackets when it has no port, and empty for the default host.
	host                   string
	treatNilableAsOptional bool
	laxDataBinding         bool
	mediaTypeSubtypePrefix string
	compression            compressionConfig
	chunking               chunkingMode
}

type compressionConfig struct {
	enable compressionMode
	// contentTypes are lower-cased; an empty list matches every Content-Type.
	contentTypes []string
}

// matches reports whether a response of contentType may be compressed: its Content-Type
// contains one of the configured content types.
func (c compressionConfig) matches(contentType string) bool {
	if len(c.contentTypes) == 0 {
		return true
	}
	contentType = strings.ToLower(contentType)
	return contentType != "" && slices.ContainsFunc(c.contentTypes, func(t string) bool {
		return strings.Contains(contentType, t)
	})
}

// defaultServiceConfig is the configuration of a service without @http:ServiceConfig, which
// also applies to the responses the dispatcher sends itself.
func defaultServiceConfig() serviceConfig {
	return serviceConfig{
		treatNilableAsOptional: true,
	}
}

func readServiceConfig(ctx *extern.Context, svcObj *values.Object) serviceConfig {
	config := defaultServiceConfig()
	annotations, _ := ctx.ObjectAnnotations(svcObj)
	annotation, ok := annotations[serviceConfigAnnotation].(*values.Map)
	if !ok {
		return config
	}
	if host, _ := annotation.Get("host"); host != nil {
		if trimmed := strings.TrimSpace(host.(string)); trimmed != defaultHost {
			config.host = strings.ToLower(trimmed)
			if _, _, err := net.SplitHostPort(config.host); err != nil {
				config.host = strings.Trim(config.host, "[]")
			}
		}
	}
	if treat, _ := annotation.Get("treatNilableAsOptional"); treat != nil {
		config.treatNilableAsOptional = treat.(bool)
	}
	if lax, _ := annotation.Get("laxDataBinding"); lax != nil {
		config.laxDataBinding = lax.(bool)
	}
	if prefix, ok := annotation.Get("mediaTypeSubtypePrefix"); ok {
		config.mediaTypeSubtypePrefix = strings.TrimSpace(prefix.(string))
	}
	if compression, ok := annotation.Get("compression"); ok {
		config.compression = readCompressionConfig(compression.(*values.Map))
	}
	switch chunking, _ := annotation.Get("chunking"); chunking {
	case "ALWAYS":
		config.chunking = chunkAlways
	case "NEVER":
		config.chunking = chunkNever
	}
	return config
}

func readCompressionConfig(compression *values.Map) compressionConfig {
	var config compressionConfig
	switch enable, _ := compression.Get("enable"); enable {
	case "ALWAYS":
		config.enable = compressAlways
	case "NEVER":
		config.enable = compressNever
	}
	if contentTypes, _ := compression.Get("contentTypes"); contentTypes != nil {
		list := contentTypes.(*values.List)
		for i := range list.Len() {
			config.contentTypes = append(config.contentTypes, strings.ToLower(strings.TrimSpace(list.Get(i).(string))))
		}
	}
	return config
}

// hostServices returns the services that answer a request whose Host header is requestHost.
// Host names compare case-insensitively. Services configured for that exact host and port come
// first, then those configured for the host name without a port, and otherwise the services
// without a host answer.
func hostServices(services []*serviceEntry, requestHost string) []*serviceEntry {
	if !slices.ContainsFunc(services, func(e *serviceEntry) bool { return e.config.host != "" }) {
		return services
	}
	requestHost = strings.ToLower(requestHost)
	name := requestHost
	if h, _, err := net.SplitHostPort(requestHost); err == nil {
		name = h
	}
	name = strings.Trim(name, "[]")
	var exact, byName, defaults []*serviceEntry
	for _, e := range services {
		switch e.config.host {
		case "":
			defaults = append(defaults, e)
		case requestHost:
			exact = append(exact, e)
		case name:
			byName = append(byName, e)
		}
	}
	switch {
	case exact != nil:
		return exact
	case byName != nil:
		return byName
	}
	return defaults
}

// javaSocketAddress writes the local address of r as Java's InetSocketAddress.toString does
// for an unresolved name: "/127.0.0.1:9090" or "/[0:0:0:0:0:0:0:1]:9090".
func javaSocketAddress(r *http.Request) string {
	addr, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return ""
	}
	host, port := splitHostPort(addr.String())
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return "/" + host + ":" + strconv.FormatInt(port, 10)
}
