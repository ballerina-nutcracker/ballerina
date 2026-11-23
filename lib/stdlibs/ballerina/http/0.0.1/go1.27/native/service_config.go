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
}

func readServiceConfig(ctx *extern.Context, svcObj *values.Object) serviceConfig {
	config := serviceConfig{treatNilableAsOptional: true}
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

// subtypePrefixWriter adds a service's media-type subtype prefix to the Content-Type of the
// response, unless the subtype already starts with it or the type is multipart.
type subtypePrefixWriter struct {
	http.ResponseWriter
	prefix      string
	wroteHeader bool
}

func (w *subtypePrefixWriter) WriteHeader(statusCode int) {
	if (statusCode >= http.StatusOK || statusCode == http.StatusSwitchingProtocols) && !w.wroteHeader {
		w.wroteHeader = true
		header := w.Header()
		mediaType, subtype, ok := strings.Cut(header.Get("Content-Type"), "/")
		if ok && mediaType != "" && !strings.EqualFold(strings.TrimSpace(mediaType), "multipart") &&
			!strings.HasPrefix(strings.ToLower(subtype), strings.ToLower(w.prefix)+"+") {
			header.Set("Content-Type", mediaType+"/"+w.prefix+"+"+subtype)
		}
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *subtypePrefixWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
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
