package main

import (
	"strings"

	"github.com/gorilla/mux"
)

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
