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
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/ballerina-nutcracker/ballerina/decimal"
	"github.com/ballerina-nutcracker/ballerina/values"
)

// corsConfig is a service's CORS configuration. Once any field is set, unset origins and
// methods default to "*", so allowOrigins is nil only when CORS is not configured.
type corsConfig struct {
	allowOrigins     []string
	allowMethods     []string
	allowHeaders     []string
	exposeHeaders    []string
	allowCredentials bool
	maxAge           int64
}

// allMethods are the methods a default resource serves, OPTIONS last.
var allMethods = []string{"GET", "HEAD", "POST", "DELETE", "PATCH", "PUT", "OPTIONS"}

func readCorsConfig(cors *values.Map) corsConfig {
	list := func(name string) []string {
		value, _ := cors.Get(name)
		items, ok := value.(*values.List)
		if !ok || items.Len() == 0 {
			return nil
		}
		result := make([]string, items.Len())
		for i := range items.Len() {
			result[i] = strings.TrimSpace(items.Get(i).(string))
		}
		return result
	}
	config := corsConfig{
		allowOrigins:  list("allowOrigins"),
		allowMethods:  list("allowMethods"),
		allowHeaders:  list("allowHeaders"),
		exposeHeaders: list("exposeHeaders"),
		maxAge:        -1,
	}
	if credentials, _ := cors.Get("allowCredentials"); credentials == true {
		config.allowCredentials = true
	}
	if maxAge, ok := cors.Get("maxAge"); ok {
		seconds, fits, _ := maxAge.(*decimal.Decimal).Int64()
		switch {
		case fits:
			config.maxAge = seconds
		case maxAge.(*decimal.Decimal).Cmp(decimal.FromInt64(0)) > 0:
			config.maxAge = math.MaxInt64
		}
	}
	configured := config.allowOrigins != nil || config.allowMethods != nil || config.allowHeaders != nil ||
		config.exposeHeaders != nil || config.allowCredentials || config.maxAge >= 0
	if configured && config.allowOrigins == nil {
		config.allowOrigins = []string{"*"}
	}
	if configured && config.allowMethods == nil {
		config.allowMethods = []string{"*"}
	}
	return config
}

func (c corsConfig) anyOrigin() bool {
	return slices.Equal(c.allowOrigins, []string{"*"})
}

// addSimpleHeaders adds the CORS headers of a resource's response to a request whose origins
// are all allowed.
func (c corsConfig) addSimpleHeaders(r *http.Request, header http.Header) {
	if c.allowOrigins == nil {
		return
	}
	addVary(header, "Origin")
	if r.Header.Get("Origin") == "" {
		return
	}
	origins := originValues(r.Header.Get("Origin"))
	if len(origins) == 0 || !c.allowsOrigins(origins) {
		return
	}
	c.addOriginHeaders(origins, header)
	if c.exposeHeaders != nil {
		header.Set("Access-Control-Expose-Headers", strings.Join(c.exposeHeaders, ", "))
	}
}

// preflightHeaders returns the headers answering a preflight request for a path whose
// resources serve methods, or nil when the request is not an allowed preflight.
func (c corsConfig) preflightHeaders(r *http.Request, methods []string) http.Header {
	origins := originValues(r.Header.Get("Origin"))
	requestMethods := listValues(r.Header.Get("Access-Control-Request-Method"))
	if c.allowOrigins == nil || len(origins) != 1 || len(requestMethods) != 1 {
		return nil
	}
	method := requestMethods[0]
	served := slices.Contains(methods, method) || method == http.MethodHead && slices.Contains(methods, http.MethodGet)
	allowed := slices.Equal(c.allowMethods, []string{"*"}) || slices.Contains(c.allowMethods, method)
	if !served || !allowed || !c.allowsOrigins(origins) {
		return nil
	}
	requestHeaders := listValues(strings.Join(r.Header.Values("Access-Control-Request-Headers"), ","))
	for _, name := range requestHeaders {
		if c.allowHeaders != nil && !slices.ContainsFunc(c.allowHeaders, func(allowed string) bool {
			return strings.EqualFold(allowed, name)
		}) {
			return nil
		}
	}
	header := http.Header{"Vary": {"Origin"}}
	c.addOriginHeaders(origins, header)
	header.Set("Access-Control-Allow-Methods", method)
	if requestHeaders != nil {
		header.Set("Access-Control-Allow-Headers", strings.Join(requestHeaders, ", "))
	}
	if c.maxAge >= 0 {
		header.Set("Access-Control-Max-Age", strconv.FormatInt(c.maxAge, 10))
	}
	return header
}

func (c corsConfig) allowsOrigins(origins []string) bool {
	return c.anyOrigin() ||
		!slices.ContainsFunc(origins, func(origin string) bool { return !slices.Contains(c.allowOrigins, origin) })
}

// addOriginHeaders allows the origins, with credentials only for listed origins.
func (c corsConfig) addOriginHeaders(origins []string, header http.Header) {
	if c.allowCredentials && !c.anyOrigin() {
		header.Set("Access-Control-Allow-Credentials", "true")
	}
	header.Set("Access-Control-Allow-Origin", strings.Join(origins, " "))
}

// originValues splits an Origin header on spaces, keeping the values that are origins.
func originValues(origin string) []string {
	var origins []string
	for value := range strings.SplitSeq(origin, " ") {
		if strings.Contains(value, "://") {
			origins = append(origins, value)
		}
	}
	return origins
}

// listValues splits a comma-separated header value into its trimmed, non-empty elements.
func listValues(value string) []string {
	var elements []string
	for element := range strings.SplitSeq(value, ",") {
		if element = strings.TrimSpace(element); element != "" {
			elements = append(elements, element)
		}
	}
	return elements
}

// allowedMethods lists the methods resources with accessors serve, a default resource serving
// every standard method: standard methods first, then custom ones sorted, then OPTIONS.
func allowedMethods(accessors []string) []string {
	serveAll := slices.Contains(accessors, "default")
	var methods []string
	for _, method := range allMethods[:len(allMethods)-1] {
		if serveAll || slices.Contains(accessors, strings.ToLower(method)) {
			methods = append(methods, method)
		}
	}
	var custom []string
	for _, accessor := range accessors {
		method := strings.ToUpper(accessor)
		if accessor != "default" && !slices.Contains(allMethods, method) {
			custom = append(custom, method)
		}
	}
	slices.Sort(custom)
	methods = append(methods, custom...)
	if len(methods) == 0 && !slices.Contains(accessors, "options") {
		return nil
	}
	return append(methods, "OPTIONS")
}
