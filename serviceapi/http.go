// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"encoding/json"
	"fmt"
	slib "github.com/mozilla/service-map/servicelib"
	"io/ioutil"
	"net/http"
)

func getPing(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(rw, "pong\n")
}

func postIndicator(rw http.ResponseWriter, req *http.Request) {
	var (
		indicator slib.RawIndicator
		err       error
	)
	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(&indicator)
	if err != nil {
		http.Error(rw, "indicator json malformed", 400)
		return
	}
	err = indicator.Validate()
	if err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	newIndicatorChan <- indicator
}

func postRRA(rw http.ResponseWriter, req *http.Request) {
	var (
		buf []byte
		err error
	)
	buf, err = ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	newrra, err := slib.NewRRA(buf)
	if err != nil {
		http.Error(rw, "rra json malformed", 400)
		return
	}
	newRRAChan <- newrra
}

func getOwners(rw http.ResponseWriter, req *http.Request) {
	obuf, err := dbGetLegacyOwners()
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	fmt.Fprintf(rw, "# name type zone operator team triageoverride\n")
	fmt.Fprintf(rw, "%v", obuf)
}
