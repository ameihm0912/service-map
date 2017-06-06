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

func newRRAInit() error {
	newRRAChan = make(chan slib.RRA, 64)
	go func() {
		var err error
		for {
			err = addRRA(<-newRRAChan)
			if err != nil {
				log.logf("error adding new rra: %v", err)
			}
		}
	}()
	return nil
}

func addRRA(rra slib.RRA) error {
	log.logf("processing new rra for %q", rra.Name)
	rawbuf, err := json.Marshal(rra.RawRRA)
	if err != nil {
		return err
	}
	return db.dbExec(`INSERT INTO rra 
		(service,
		impact_availrep, impact_availprd, impact_availfin,
		impact_integrep, impact_integprd, impact_integfin,
		impact_confirep, impact_confiprd, impact_confifin,
		prob_availrep, prob_availprd, prob_availfin,
		prob_integrep, prob_integprd, prob_integfin,
		prob_confirep, prob_confiprd, prob_confifin,
		datadefault, lastupdated, timestamp, raw)
		VALUES ($1,
		$2, $3, $4,
		$5, $6, $7,
		$8, $9, $10,
		$11, $12, $13,
		$14, $15, $16,
		$17, $18, $19,
		$20, $21, now(), $22)`,
		rra.Name,
		rra.AvailRepImpact, rra.AvailPrdImpact, rra.AvailFinImpact,
		rra.IntegRepImpact, rra.IntegPrdImpact, rra.IntegFinImpact,
		rra.ConfiRepImpact, rra.ConfiPrdImpact, rra.ConfiFinImpact,
		rra.AvailRepProb, rra.AvailPrdProb, rra.AvailFinProb,
		rra.IntegRepProb, rra.IntegPrdProb, rra.IntegFinProb,
		rra.ConfiRepProb, rra.ConfiPrdProb, rra.ConfiFinProb,
		rra.DefData, rra.LastUpdated, rawbuf)
}
