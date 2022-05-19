package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync"
	"time"

	"github.com/decred/dcrlnlpd/internal/version"
	"github.com/decred/dcrlnlpd/server"
	"github.com/gorilla/mux"
)

func indexHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "%s\n%s\n", appName, version.String())
}

func handler(s *server.Server) http.Handler {
	router := mux.NewRouter().StrictSlash(true)
	router.Methods("GET").Path("/").Name("index").HandlerFunc(indexHandler)
	server.NewV1Handler(s, router)
	logRouterConfig(router)
	return router
}

func _main() error {
	// Load configuration and parse command line.  This function also
	// initializes logging and configures it accordingly.
	cfg, _, err := loadConfig()
	if err != nil {
		return err
	}

	// Start the main context for the app
	ctx := shutdownListener()
	var wg sync.WaitGroup

	log.Infof("Initing %s v%s on %s", appName, version.String(), cfg.activeNet)

	// Enable http profiling server if requested.
	if cfg.Profile != "" {
		go func() {
			listenAddr := cfg.Profile
			log.Infof("Creating profiling server "+
				"listening on %s", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			err := http.ListenAndServe(listenAddr, nil)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Errorf("Fatal error running the http profiler: %v", err)
				requestShutdown()
			}
		}()
	}

	// Create and start the server server.
	svrCfg, err := cfg.serverConfig()
	if err != nil {
		return err
	}
	drsvr, err := server.New(svrCfg)
	if err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		// Errors other than context canceled are fatal.
		err := drsvr.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Errorf("Fatal error running the server instance: %v", err)
			requestShutdown()
		}
		wg.Done()
	}()

	// Create and start all http listening interfaces.
	svr := &http.Server{
		Handler: handler(drsvr),
	}
	listeners, err := cfg.listeners()
	if err != nil {
		log.Error(err.Error())
		return err
	}
	for _, l := range listeners {
		wg.Add(1)
		go func(l net.Listener) {
			log.Infof("Listening on %s", l.Addr().String())
			svr.Serve(l)
			wg.Done()
		}(l)
	}

	// Wait until the app is commanded to shutdown to close the server.
	<-ctx.Done()

	// Wait up to 5 seconds until all connections are gracefully closed
	// before terminating the process.
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err = svr.Shutdown(timeout)
	cancel() // Prevent context leak.
	if errors.Is(err, context.Canceled) {
		log.Warnf("Terminating process before all connections were done")
	}

	// Wait for all goroutines to finish.
	wg.Wait()

	return nil
}

func main() {
	if err := _main(); err != nil && err != errCmdDone {
		fmt.Println(err)
		os.Exit(1)
	}
}
