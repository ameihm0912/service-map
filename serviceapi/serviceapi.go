// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	slib "github.com/mozilla/service-map/servicelib"
	gcfg "gopkg.in/gcfg.v1"
	"net/http"
	"os"
	"sync"
	"time"
)

type config struct {
	Interlink struct {
		RulesPath string
	}
	Database struct {
		Host     string
		Database string
		Username string
		Password string
	}
}

func (c *config) validate() error {
	return nil
}

var (
	cfg              config
	log              logger
	newIndicatorChan chan slib.RawIndicator
	newRRAChan       chan slib.RRA
	wg               sync.WaitGroup
)

type logger struct {
	logChan chan string
}

func (l *logger) logf(s string, args ...interface{}) {
	buf := fmt.Sprintf(s, args...)
	tstr := time.Now().Format("2006-01-02 15:04:05")
	l.logChan <- fmt.Sprintf("[%v] %v", tstr, buf)
}

func newLogger() (ret logger, err error) {
	ret.logChan = make(chan string, 64)
	go func() {
		defer wg.Done()
		for {
			l, ok := <-ret.logChan
			if !ok {
				break
			}
			fmt.Print(l + "\n")
		}
	}()
	return
}

func doExit(ret int) {
	close(log.logChan)
	wg.Wait()
	os.Exit(ret)
}

func apiWrapper(h func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		log.logf("api: %v %v %v %v", req.RemoteAddr, req.Method, req.URL, req.Proto)
		h(rw, req)
	}
}

func main() {
	var cfgpath string

	flag.StringVar(&cfgpath, "f", "", "path to configuration file")
	flag.Parse()

	if cfgpath == "" {
		fmt.Fprintf(os.Stderr, "error: must specify configuration file with -f\n")
		os.Exit(1)
	}
	err := gcfg.ReadFileInto(&cfg, cfgpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %v: %v\n", cfgpath, err)
		os.Exit(1)
	}
	err = cfg.validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error validating %v: %v\n", cfgpath, err)
		os.Exit(1)
	}
	log, err = newLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing logging: %v\n", err)
		os.Exit(1)
	}
	wg.Add(1)
	err = dbInit(cfg)
	if err != nil {
		log.logf("error initializing database: %v\n", err)
		doExit(1)
	}
	err = newIndicatorInit()
	if err != nil {
		log.logf("error initializing new indicator processor: %v\n", err)
		doExit(1)
	}
	err = newRRAInit()
	if err != nil {
		log.logf("error initializing new rra processor: %v\n", err)
		doExit(1)
	}
	err = interlinkInit()
	if err != nil {
		log.logf("error initializing interlink system: %v\n", err)
		doExit(1)
	}
	r := mux.NewRouter()
	s := r.PathPrefix("/api/v1").Subrouter()
	s.HandleFunc("/indicator", apiWrapper(postIndicator)).Methods("POST")
	s.HandleFunc("/ping", apiWrapper(getPing)).Methods("GET")
	s.HandleFunc("/rra", apiWrapper(postRRA)).Methods("POST")
	s.HandleFunc("/owners", apiWrapper(getOwners)).Methods("GET")
	err = http.ListenAndServe(":8080", context.ClearHandler(r))
	if err != nil {
		log.logf("http.ListenAndServe: %v", err)
		doExit(1)
	}
	doExit(0)
}
