// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package servicelib

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Describes an RRA, as stored by service-map
type RRA struct {
	Name        string    `json:"name,omitempty"`
	LastUpdated time.Time `json:"lastupdated"`
	ID          int       `json:"id,omitempty"`

	/* Attribute impact scores */
	AvailRepImpact string `json:"availability_reputation_impact,omitempty"`
	AvailPrdImpact string `json:"availability_productivity_impact,omitempty"`
	AvailFinImpact string `json:"availability_financial_impact,omitempty"`
	IntegRepImpact string `json:"integrity_reputation_impact,omitempty"`
	IntegPrdImpact string `json:"integrity_productivity_impact,omitempty"`
	IntegFinImpact string `json:"integrity_financial_impact,omitempty"`
	ConfiRepImpact string `json:"confidentiality_reputation_impact,omitempty"`
	ConfiPrdImpact string `json:"confidentiality_productivity_impact,omitempty"`
	ConfiFinImpact string `json:"confidentiality_financial_impact,omitempty"`

	/* Attribute probability scores */
	AvailRepProb string `json:"availability_reputation_probability,omitempty"`
	AvailPrdProb string `json:"availability_productivity_probability,omitempty"`
	AvailFinProb string `json:"availability_financial_probability,omitempty"`
	IntegRepProb string `json:"integrity_reputation_probability,omitempty"`
	IntegPrdProb string `json:"integrity_productivity_probability,omitempty"`
	IntegFinProb string `json:"integrity_financial_probability,omitempty"`
	ConfiRepProb string `json:"confidentiality_reputation_probability,omitempty"`
	ConfiPrdProb string `json:"confidentiality_productivity_probability,omitempty"`
	ConfiFinProb string `json:"confidentiality_financial_probability,omitempty"`

	DefData string `json:"default_data_classification,omitempty"`

	RawRRA interface{} `json:"rra_details,omitempty"` // The original raw RRA
}

func (r *RRA) Validate() error {
	if r.Name == "" {
		return errors.New("rra must have a name")
	}
	return nil
}

// For a given RRA, return the impact and probability value for the highest
// risk scenario (according to the RRA) related to reputation.
func (r *RRA) HighestRiskReputation() (float64, float64) {
	// XXX Assumed values have been normalized here
	repavi, _ := ImpactValueFromLabel(r.AvailRepImpact)
	repavp, _ := ImpactValueFromLabel(r.AvailRepProb)
	repcfi, _ := ImpactValueFromLabel(r.ConfiRepImpact)
	repcfp, _ := ImpactValueFromLabel(r.ConfiRepProb)
	repiti, _ := ImpactValueFromLabel(r.IntegRepImpact)
	repitp, _ := ImpactValueFromLabel(r.IntegRepProb)
	rskav := repavi * repavp
	rskcf := repcfi * repcfp
	rskit := repiti * repitp
	var candr, candi, candp *float64
	candi = &repavi
	candp = &repavp
	candr = &rskav
	if rskcf > *candr {
		candi = &repcfi
		candp = &repcfp
		candr = &rskcf
	}
	if rskit > *candr {
		candi = &repiti
		candp = &repitp
		candr = &rskit
	}
	return *candi, *candp
}

// For a given RRA, return the impact and probability value for the highest
// risk scenario (according to the RRA) related to productivity.
func (r *RRA) HighestRiskProductivity() (float64, float64) {
	// XXX Assumed values have been normalized here
	prdavi, _ := ImpactValueFromLabel(r.AvailPrdImpact)
	prdavp, _ := ImpactValueFromLabel(r.AvailPrdProb)
	prdcfi, _ := ImpactValueFromLabel(r.ConfiPrdImpact)
	prdcfp, _ := ImpactValueFromLabel(r.ConfiPrdProb)
	prditi, _ := ImpactValueFromLabel(r.IntegPrdImpact)
	prditp, _ := ImpactValueFromLabel(r.IntegPrdProb)
	rskav := prdavi * prdavp
	rskcf := prdcfi * prdcfp
	rskit := prditi * prditp
	var candr, candi, candp *float64
	candi = &prdavi
	candp = &prdavp
	candr = &rskav
	if rskcf > *candr {
		candi = &prdcfi
		candp = &prdcfp
		candr = &rskcf
	}
	if rskit > *candr {
		candi = &prditi
		candp = &prditp
		candr = &rskit
	}
	return *candi, *candp
}

// For a given RRA, return the impact and probability value for the highest
// risk scenario (according to the RRA) related to finance.
func (r *RRA) HighestRiskFinancial() (float64, float64) {
	// XXX Assumed values have been normalized here
	finavi, _ := ImpactValueFromLabel(r.AvailFinImpact)
	finavp, _ := ImpactValueFromLabel(r.AvailFinProb)
	fincfi, _ := ImpactValueFromLabel(r.ConfiFinImpact)
	fincfp, _ := ImpactValueFromLabel(r.ConfiFinProb)
	finiti, _ := ImpactValueFromLabel(r.IntegFinImpact)
	finitp, _ := ImpactValueFromLabel(r.IntegFinProb)
	rskav := finavi * finavp
	rskcf := fincfi * fincfp
	rskit := finiti * finitp
	var candr, candi, candp *float64
	candi = &finavi
	candp = &finavp
	candr = &rskav
	if rskcf > *candr {
		candi = &fincfi
		candp = &fincfp
		candr = &rskcf
	}
	if rskit > *candr {
		candi = &finiti
		candp = &finitp
		candr = &rskit
	}
	return *candi, *candp
}

// Score values per impact label
const (
	ImpactUnknownValue = 0.0
	ImpactLowValue     = 1.0
	ImpactMediumValue  = 2.0
	ImpactHighValue    = 3.0
	ImpactMaxValue     = 4.0
)

// Values for data classification
const (
	DataUnknownValue = 0.0
	DataPublicValue  = 1.0
	DataConfIntValue = 2.0
	DataConfResValue = 3.0
	DataConfSecValue = 4.0
)

type RRAAttribute struct {
	Attribute   string  `json:"attribute"`
	Impact      float64 `json:"impact"`
	Probability float64 `json:"probability"`
}

// NewRRA converts a byte buffer containing a raw RRA JSON document (e.g.,
// published by rra2json) and converts it, returning an RRA type.
func NewRRA(buf []byte) (ret RRA, err error) {
	var raw RawRRA
	err = json.Unmarshal(buf, &raw)
	if err != nil {
		return
	}
	// Validate the input RRA, this will also do any normalization that is required
	err = raw.Validate()
	if err != nil {
		return
	}

	ret.Name = raw.Details.Metadata.Service
	ret.DefData = raw.Details.Data.Default
	ret.LastUpdated = raw.LastModified

	ret.AvailRepImpact = raw.Details.Risk.Availability.Reputation.Impact
	ret.AvailPrdImpact = raw.Details.Risk.Availability.Productivity.Impact
	ret.AvailFinImpact = raw.Details.Risk.Availability.Finances.Impact
	ret.IntegRepImpact = raw.Details.Risk.Integrity.Reputation.Impact
	ret.IntegPrdImpact = raw.Details.Risk.Integrity.Productivity.Impact
	ret.IntegFinImpact = raw.Details.Risk.Integrity.Finances.Impact
	ret.ConfiRepImpact = raw.Details.Risk.Confidentiality.Reputation.Impact
	ret.ConfiPrdImpact = raw.Details.Risk.Confidentiality.Productivity.Impact
	ret.ConfiFinImpact = raw.Details.Risk.Confidentiality.Finances.Impact

	ret.AvailRepProb = raw.Details.Risk.Availability.Reputation.Probability
	ret.AvailPrdProb = raw.Details.Risk.Availability.Productivity.Probability
	ret.AvailFinProb = raw.Details.Risk.Availability.Finances.Probability
	ret.IntegRepProb = raw.Details.Risk.Integrity.Reputation.Probability
	ret.IntegPrdProb = raw.Details.Risk.Integrity.Productivity.Probability
	ret.IntegFinProb = raw.Details.Risk.Integrity.Finances.Probability
	ret.ConfiRepProb = raw.Details.Risk.Confidentiality.Reputation.Probability
	ret.ConfiPrdProb = raw.Details.Risk.Confidentiality.Productivity.Probability
	ret.ConfiFinProb = raw.Details.Risk.Confidentiality.Finances.Probability

	err = json.Unmarshal(buf, &ret.RawRRA)
	if err != nil {
		return
	}

	return
}

// Describes calculated risk for a service, based on an RRA and known
// data points
type RRARisk struct {
	RRA RRA `json:"rra"` // The RRA we are describing

	// The attribute from the RRA we use as the basis for risk calculations
	// (business impact) for the service. For example, this could be "reputation",
	// "productivity", or "financial" depending on which attribute in the RRA
	// yields the highest defined risk.
	//
	// Previous versions of this would generate scenarios across all attributes
	// in the RRA. This behavior is not necessarily desirable as it can end up
	// devaluing high impact attributes when we consider the entire set combined.
	UsedRRAAttrib RRAAttribute

	Risk struct {
		WorstCase      float64 `json:"worst_case"`
		WorstCaseLabel string  `json:"worst_case_label"`
		Median         float64 `json:"median"`
		MedianLabel    string  `json:"median_label"`
		Average        float64 `json:"average"`
		AverageLabel   string  `json:"average_label"`
		DataClass      float64 `json:"data_classification"`
		Impact         float64 `json:"highest_business_impact"`
		ImpactLabel    string  `json:"highest_business_impact_label"`
	} `json:"risk"`

	Scenarios []RiskScenario `json:"scenarios"` // Risk scenarios
}

func (r *RRARisk) Validate() error {
	err := r.RRA.Validate()
	if err != nil {
		return err
	}
	return nil
}

// Stores information used to support probability for risk calculation; this
// generally would be created using control information and is combined with the
// RRA impact scores to produce estimated service risk
type RiskScenario struct {
	Name        string  `json:"name"` // Name describing the datapoint
	Probability float64 `json:"probability"`
	Impact      float64 `json:"impact"`
	Score       float64 `json:"score"`
	NoData      bool    `json:"nodata"`   // No data exists for proper calculation
	Coverage    string  `json:"coverage"` // Coverage (partial, complete, none, unknown)
}

// Validates a RiskScenario for consistency
func (r *RiskScenario) Validate() error {
	if r.Name == "" {
		return errors.New("scenario must have a name")
	}
	if r.Coverage != "none" && r.Coverage != "partial" &&
		r.Coverage != "complete" && r.Coverage != "unknown" {
		return fmt.Errorf("scenario \"%v\" coverage invalid \"%v\"", r.Name, r.Coverage)
	}
	return nil
}

// Convert an impact label into the numeric representation from 1 - 4 for
// that label
func ImpactValueFromLabel(l string) (float64, error) {
	switch l {
	case "maximum":
		return ImpactMaxValue, nil
	case "high":
		return ImpactHighValue, nil
	case "medium":
		return ImpactMediumValue, nil
	case "low":
		return ImpactLowValue, nil
	case "unknown":
		// XXX Return low here if the value is set to unknown to handle older
		// format RRAs and still use the data in risk calculation.
		return ImpactLowValue, nil
	}
	return 0, fmt.Errorf("invalid impact label %v", l)
}

// Convert an impact label into the numeric representation from 1 - 4 for
// that label
func DataValueFromLabel(l string) (float64, error) {
	switch l {
	case "confidential secret":
		return DataConfSecValue, nil
	case "confidential restricted":
		return DataConfResValue, nil
	case "confidential internal":
		return DataConfIntValue, nil
	case "public":
		return DataPublicValue, nil
	case "unknown":
		return DataUnknownValue, nil
	}
	return 0, fmt.Errorf("invalid impact label %v", l)
}

// Sanitize an impact label and verify it's a valid value
func SanitizeImpactLabel(l string) (ret string, err error) {
	if l == "" {
		err = fmt.Errorf("invalid zero length label")
		return
	}
	ret = strings.ToLower(l)
	if ret != "maximum" && ret != "high" && ret != "medium" &&
		ret != "low" && ret != "unknown" {
		err = fmt.Errorf("invalid impact label \"%v\"", ret)
	}
	return
}

// Covert an impact value from 1 - 4 to the string value for that label,
// note that does not handle decimal values in the floating point value
// and should only be used with 1.0, 2.0, 3.0, or 4.0
func ImpactLabelFromValue(v float64) (string, error) {
	switch v {
	case ImpactMaxValue:
		return "maximum", nil
	case ImpactHighValue:
		return "high", nil
	case ImpactMediumValue:
		return "medium", nil
	case ImpactLowValue:
		return "low", nil
	case ImpactUnknownValue:
		return "unknown", nil
	}
	return "", fmt.Errorf("invalid impact value %v", v)
}

// Given a risk score from 1 - 16, convert that score into
// the string value that represents the risk
func NormalLabelFromValue(v float64) string {
	if v >= 13 {
		return "maximum"
	} else if v >= 9 {
		return "high"
	} else if v >= 5 {
		return "medium"
	}
	return "low"
}

// RawRRA defines the structure expected as input when a new RRA is posted to
// service-map. More specifically, this would be the RRA json as is submitted
// to service-map from rra2json.
//
// We do not define all fields that would be present in that input document, but
// limit it to the fields we actually make use of for risk processing.
type RawRRA struct {
	Details      RawRRADetails `json:"details"`
	LastModified time.Time     `json:"lastmodified"`
}

func (r *RawRRA) Validate() error {
	return r.Details.Validate()
}

type RawRRADetails struct {
	Metadata RawRRAMetadata `json:"metadata"`
	Risk     RawRRARisk     `json:"risk"`
	Data     RawRRAData     `json:"data"`
}

func (r *RawRRADetails) Validate() error {
	err := r.Metadata.Validate()
	if err != nil {
		return err
	}
	err = r.Risk.Validate()
	if err != nil {
		return fmt.Errorf("%q: %v", r.Metadata.Service, err)
	}
	err = r.Data.Validate()
	if err != nil {
		return fmt.Errorf("%q: %v", r.Metadata.Service, err)
	}
	return nil
}

type RawRRAMetadata struct {
	Service string `json:"service"`
}

func (r *RawRRAMetadata) Validate() error {
	if r.Service == "" {
		return errors.New("rra has no service name")
	}
	// Do some sanitization of the service name if necessary
	r.Service = strings.Replace(r.Service, "\n", " ", -1)
	r.Service = strings.TrimSpace(r.Service)
	return nil
}

type RawRRAData struct {
	Default string `json:"default"`
}

func (r *RawRRAData) Validate() error {
	if r.Default == "" {
		return errors.New("rra has no default data classification")
	}
	// Sanitize the data classification
	// XXX This should likely be checked against a list of known valid
	// strings, and we just reject importing an RRA that has a data
	// classification value we don't know about.
	r.Default = strings.ToLower(r.Default)
	// Convert from some older classification values
	switch r.Default {
	case "internal":
		r.Default = "confidential internal"
	case "restricted":
		r.Default = "confidential restricted"
	case "secret":
		r.Default = "confidential secret"
	}
	return nil
}

type RawRRARisk struct {
	Confidentiality RawRRARiskAttr `json:"confidentiality"`
	Integrity       RawRRARiskAttr `json:"integrity"`
	Availability    RawRRARiskAttr `json:"availability"`
}

func (r *RawRRARisk) Validate() error {
	err := r.Confidentiality.Validate()
	if err != nil {
		return err
	}
	err = r.Integrity.Validate()
	if err != nil {
		return err
	}
	err = r.Availability.Validate()
	if err != nil {
		return err
	}
	return nil
}

type RawRRARiskAttr struct {
	Reputation   RawRRAMeasure `json:"reputation"`
	Finances     RawRRAMeasure `json:"finances"`
	Productivity RawRRAMeasure `json:"productivity"`
}

func (r *RawRRARiskAttr) Validate() error {
	err := r.Reputation.Validate()
	if err != nil {
		return err
	}
	err = r.Finances.Validate()
	if err != nil {
		return err
	}
	err = r.Productivity.Validate()
	if err != nil {
		return err
	}
	return nil
}

type RawRRAMeasure struct {
	Impact      string `json:"impact"`
	Probability string `json:"probability"`
}

func (r *RawRRAMeasure) Validate() (err error) {
	r.Impact, err = SanitizeImpactLabel(r.Impact)
	if err != nil {
		return err
	}
	// XXX If the probability value is unset, just default it to unknown
	// here and continue. We can proceed without this value, if we at least
	// have the impact. Without this though certain calculation datapoints
	// may not be possible.
	if r.Probability == "" {
		r.Probability = "unknown"
	}
	r.Probability, err = SanitizeImpactLabel(r.Probability)
	if err != nil {
		return err
	}
	return nil
}
