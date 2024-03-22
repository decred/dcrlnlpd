package main

import (
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func requestAddr(req *http.Request) string {
	res := req.RemoteAddr
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		if values, ok := req.Header[header]; ok {
			for _, value := range values {
				res += strings.TrimSpace(value)
			}
		}
	}

	return res
}

// isReqFromLocalhost checks if the request came from localhost. It checks the
// X-Forwarded-For and X-Real-IP headers first and then falls back to the
// RemoteAddr if necessary.
func isReqFromLocalhost(req *http.Request) bool {
	// Function to check if an IP from a given string is a loopback address.
	// It splits the host and port if necessary and checks the IP.
	isLoopback := func(addr string) bool {
		ip := net.ParseIP(addr)
		return ip != nil && ip.IsLoopback()
	}

	// Check X-Forwarded-For and X-Real-IP headers for the original client IP
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		if values, ok := req.Header[header]; ok {
			for _, value := range values {
				for _, ipStr := range strings.Split(value, ",") {
					ipStr = strings.TrimSpace(ipStr)
					if isLoopback(ipStr) {
						return true
					}
				}
			}
		}
	}

	// If the headers are not present, fall back to the RemoteAddr.
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		host = req.RemoteAddr
	}
	return isLoopback(host)
}

func logRouterConfig(r *mux.Router) {
	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			log.Debugf("ROUTE: %s", pathTemplate)
		}
		pathRegexp, err := route.GetPathRegexp()
		if err == nil {
			log.Debugf("Path regexp: %s", pathRegexp)

		}
		queriesTemplates, err := route.GetQueriesTemplates()
		if err == nil {
			log.Debugf("Queries templates: %s", strings.Join(queriesTemplates, ","))

		}
		queriesRegexps, err := route.GetQueriesRegexp()
		if err == nil {
			log.Debugf("Queries regexps: %s", strings.Join(queriesRegexps, ","))

		}
		methods, err := route.GetMethods()
		if err == nil {
			log.Debugf("Methods: %s", strings.Join(methods, ","))

		}
		log.Debugf("")
		return nil
	})
	if err != nil {
		panic(err)
	}
}
