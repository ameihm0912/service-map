// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package servicelib

import (
	"errors"
	"strings"
	"time"
)

type Indicator struct {
	Type        string      `json:"asset_type,omitempty"`           // Asset type e.g., hostname, website, etc
	Name        string      `json:"asset_identifier,omitempty"`     // Name of asset
	Zone        string      `json:"zone,omitempty"`                 // Zone asset identified in
	Description string      `json:"description,omitempty"`          // Description for asset
	Timestamp   time.Time   `json:"timestamp_utc"`                  // Timestamp indicatator is relevant for
	EventSource string      `json:"event_source_name,omitempty"`    // Source of indicator
	Likelihood  string      `json:"likelihood_indicator,omitempty"` // Likelihood indicator value
	Details     interface{} `json:"details,omitempty"`              // Free form details sub structure
}

func (i *Indicator) Validate() error {
	if i.Type == "" {
		return errors.New("indicator asset type missing")
	}
	if i.Name == "" {
		return errors.New("indicator asset name/identifier missing")
	}
	if i.EventSource == "" {
		return errors.New("indicator event source missing")
	}
	i.Likelihood = strings.ToLower(i.Likelihood)
	switch i.Likelihood {
	case "maximum":
	case "high":
	case "medium":
	case "low":
	default:
		return errors.New("indicator has invalid likelihood value")
	}
	return nil
}
