// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"encoding/json"
	slib "github.com/mozilla/service-map/servicelib"
)

func newIndicatorInit() error {
	newIndicatorChan = make(chan slib.Indicator, 64)
	go func() {
		var err error
		for {
			err = addIndicator(<-newIndicatorChan)
			if err != nil {
				log.logf("error adding new indicator: %v", err)
			}
		}
	}()
	return nil
}

func addIndicator(indicator slib.Indicator) error {
	log.logf("processing new indicator for %q, type %q", indicator.Name, indicator.Type)
	detailsbuf, err := json.Marshal(indicator.Details)
	if err != nil {
		return err
	}
	return db.dbExec(`INSERT INTO asset
		(assettype, name, zone, description, timestamp, event_source,
		likelihood_indicator, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		indicator.Type, indicator.Name, indicator.Zone, indicator.Description,
		indicator.Timestamp, indicator.EventSource, indicator.Likelihood,
		string(detailsbuf))
}
