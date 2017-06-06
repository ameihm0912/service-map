// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

// Related to the interlink rule set parser and application of the rules

import (
	"bufio"
	"errors"
	slib "github.com/mozilla/service-map/servicelib"
	"os"
	"strings"
	"time"
)

// Rule types
const (
	_ = iota
	ASSETGROUP_ADD
	ASSETGROUP_LINK_SERVICE
	HOST_LINK_ASSETGROUP
	WEBSITE_ADD
	WEBSITE_LINK_ASSETGROUP
	HOST_OWNERSHIP
	OWNER_ADD
)

// Defines a rule in the interlink system, rules are parsed from a file into a slice
// of the interlinkRule type, and subsequently applied to the database
type interlinkRule struct {
	ruletype int // Rule type (e.g., ASSETGROUP_ADD, etc)

	srcHostMatch       string
	srcAssetGroupMatch string

	destServiceMatch    string
	destAssetGroupMatch string

	srcWebsiteMatch  string
	destWebsiteMatch string

	destOwnerMatch struct {
		Operator string
		Team     string
	}

	destTriageOverride string
}

// Runs all interlink rules in the required order
func interlinkRunRules(rules []interlinkRule) error {
	log.logf("interlink rules running")
	tx, err := dbNewTx()
	if err != nil {
		return err
	}
	// First, add any asset groups
	err = interlinkAddAssetGroups(tx, rules)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Purge any asset groups present in the system that are no longer
	// required
	err = interlinkRemoveAssetGroups(tx, rules)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Host asset group linkage
	err = interlinkHostLinkAssetGroups(tx, rules)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Asset group RRA linkage
	err = interlinkAssetGroupLinkRRA(tx, rules)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Add owners
	err = interlinkAddOwners(tx, rules)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Purge any owners present in the system that are no longer
	// required
	err = interlinkRemoveOwners(tx, rules)
	if err != nil {
		tx.Rollback()
		return err
	}
	// Set ownership on indicators/assets
	err = interlinkOwnership(tx, rules)
	if err != nil {
		tx.Rollback()
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	log.logf("interlink rules completed")
	return nil
}

// Assign owners
func interlinkOwnership(tx dbTx, rules []interlinkRule) error {
	var err error
	for _, rule := range rules {
		if rule.ruletype != HOST_OWNERSHIP {
			continue
		}
		err = dbHostOwnership(&tx, rule.srcHostMatch, rule.destOwnerMatch.Team,
			rule.destOwnerMatch.Operator, rule.destTriageOverride)
		if err != nil {
			return err
		}
	}
	return nil
}

// Adds any asset groups to the database that are missing
func interlinkAddOwners(tx dbTx, rules []interlinkRule) error {
	var err error
	for _, rule := range rules {
		if rule.ruletype != OWNER_ADD {
			continue
		}
		newown := slib.Owner{
			Team:     rule.destOwnerMatch.Team,
			Operator: rule.destOwnerMatch.Operator,
		}
		err = dbAddOwner(&tx, newown)
		if err != nil {
			return err
		}
	}
	return nil
}

// Adds any asset groups to the database that are missing
func interlinkAddAssetGroups(tx dbTx, rules []interlinkRule) error {
	var err error
	for _, rule := range rules {
		if rule.ruletype != ASSETGROUP_ADD {
			continue
		}
		newgrp := slib.AssetGroup{Name: rule.destAssetGroupMatch}
		err = dbAddAssetGroup(&tx, newgrp)
		if err != nil {
			return err
		}
	}
	return nil
}

// Remove any owners from the database that are no longer valid
func interlinkRemoveOwners(tx dbTx, rules []interlinkRule) error {
	var (
		err    error
		owners []slib.Owner
	)
	for _, rule := range rules {
		if rule.ruletype != OWNER_ADD {
			continue
		}
		own := slib.Owner{
			Team:     rule.destOwnerMatch.Team,
			Operator: rule.destOwnerMatch.Operator,
		}
		owners = append(owners, own)
	}
	dbowners, err := dbGetOwners(&tx)
	if err != nil {
		return err
	}
	for _, own := range dbowners {
		found := false
		for _, x := range owners {
			if x.Team == own.Team && x.Operator == own.Operator {
				found = true
				break
			}
		}
		if !found {
			err = dbRemoveOwner(&tx, own)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Remove any asset groups from the database that are no longer valid
func interlinkRemoveAssetGroups(tx dbTx, rules []interlinkRule) error {
	var (
		err    error
		groups []string
	)
	for _, rule := range rules {
		if rule.ruletype != ASSETGROUP_ADD {
			continue
		}
		groups = append(groups, rule.destAssetGroupMatch)
	}
	dbgroups, err := dbGetAssetGroups(&tx)
	if err != nil {
		return err
	}
	for _, grp := range dbgroups {
		found := false
		for _, x := range groups {
			if x == grp.Name {
				found = true
				break
			}
		}
		if !found {
			err = dbRemoveAssetGroup(&tx, grp)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Link host type assets to the asset group they belong in
func interlinkHostLinkAssetGroups(tx dbTx, rules []interlinkRule) error {
	var (
		err  error
		grps []string
	)
	for _, rule := range rules {
		if rule.ruletype != HOST_LINK_ASSETGROUP {
			continue
		}
		found := false
		for _, x := range grps {
			if x == rule.destAssetGroupMatch {
				found = true
				break
			}
		}
		if !found {
			err = dbPurgeAssetGroupHosts(&tx, rule.destAssetGroupMatch)
			if err != nil {
				return err
			}
			grps = append(grps, rule.destAssetGroupMatch)
		}
		err = dbHostLinkAssetGroup(&tx, rule.srcHostMatch, rule.destAssetGroupMatch)
		if err != nil {
			return err
		}
	}
	return nil
}

// Link asset groups to the applicable RRA services
func interlinkAssetGroupLinkRRA(tx dbTx, rules []interlinkRule) error {
	var (
		err  error
		grps []string
	)
	for _, rule := range rules {
		if rule.ruletype != ASSETGROUP_LINK_SERVICE {
			continue
		}
		found := false
		for _, x := range grps {
			if x == rule.srcAssetGroupMatch {
				found = true
				break
			}
		}
		if !found {
			err = dbPurgeAssetGroupRRAs(&tx, rule.srcAssetGroupMatch)
			if err != nil {
				return err
			}
			grps = append(grps, rule.srcAssetGroupMatch)
		}
		err = dbAssetGroupLinkRRA(&tx, rule.srcAssetGroupMatch, rule.destServiceMatch)
		if err != nil {
			return err
		}
	}
	return nil
}

// Loads interlink rules from path, returning a slice of the processed rules
func interlinkLoadRules(path string) ([]interlinkRule, error) {
	var (
		rules []interlinkRule
		err   error
	)

	fd, err := os.Open(path)
	if err != nil {
		return rules, err
	}
	defer fd.Close()

	scnr := bufio.NewScanner(fd)
	for scnr.Scan() {
		buf := scnr.Text()
		if len(buf) == 0 {
			continue
		}
		tokens := strings.Split(buf, " ")

		if len(tokens) > 0 && tokens[0][0] == '#' {
			continue
		}

		var nr interlinkRule
		if len(tokens) < 2 {
			return rules, errors.New("interlink rule without enough arguments")
		}
		valid := false
		if len(tokens) == 3 && tokens[0] == "add" && tokens[1] == "assetgroup" {
			nr.ruletype = ASSETGROUP_ADD
			nr.destAssetGroupMatch = tokens[2]
			valid = true
		} else if len(tokens) == 4 && tokens[0] == "add" && tokens[1] == "owner" {
			nr.ruletype = OWNER_ADD
			nr.destOwnerMatch.Operator = tokens[2]
			nr.destOwnerMatch.Team = tokens[3]
			valid = true
		} else if len(tokens) == 3 && tokens[0] == "add" && tokens[1] == "website" {
			nr.ruletype = WEBSITE_ADD
			nr.destWebsiteMatch = tokens[2]
			valid = true
		} else if len(tokens) == 6 && tokens[0] == "assetgroup" &&
			tokens[1] == "matches" && tokens[3] == "link" && tokens[4] == "service" {
			nr.ruletype = ASSETGROUP_LINK_SERVICE
			nr.srcAssetGroupMatch = tokens[2]
			nr.destServiceMatch = tokens[5]
			valid = true
		} else if len(tokens) == 6 && tokens[0] == "host" &&
			tokens[1] == "matches" && tokens[3] == "link" && tokens[4] == "assetgroup" {
			nr.ruletype = HOST_LINK_ASSETGROUP
			nr.srcHostMatch = tokens[2]
			nr.destAssetGroupMatch = tokens[5]
			valid = true
		} else if len(tokens) >= 6 && tokens[0] == "host" &&
			tokens[1] == "matches" && tokens[3] == "ownership" {
			nr.ruletype = HOST_OWNERSHIP
			nr.srcHostMatch = tokens[2]
			nr.destOwnerMatch.Operator = tokens[4]
			nr.destOwnerMatch.Team = tokens[5]
			if len(tokens) == 7 {
				nr.destTriageOverride = tokens[6]
			}
			valid = true
		} else if len(tokens) == 6 && tokens[0] == "website" &&
			tokens[1] == "matches" && tokens[3] == "link" && tokens[4] == "assetgroup" {
			nr.ruletype = WEBSITE_LINK_ASSETGROUP
			nr.srcWebsiteMatch = tokens[2]
			nr.destAssetGroupMatch = tokens[5]
			valid = true
		}
		if !valid {
			return rules, errors.New("syntax error in interlink rules")
		}
		rules = append(rules, nr)
	}

	log.logf("interlink loaded %v rules", len(rules))
	return rules, nil
}

func interlinkInit() error {
	go func() {
		for {
			time.Sleep(time.Second * 5)
			rules, err := interlinkLoadRules(cfg.Interlink.RulesPath)
			if err != nil {
				log.logf("error loading interlink rules: %v", err)
				continue
			}
			err = interlinkRunRules(rules)
			if err != nil {
				log.logf("error running interlink rules: %v", err)
			}
		}
	}()
	return nil
}
