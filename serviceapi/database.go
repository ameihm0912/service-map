// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	slib "github.com/mozilla/service-map/servicelib"
)

var db dbConn

type dbInterface interface {
	dbExec(string, ...interface{}) error
	dbQuery(string, ...interface{}) (*sql.Rows, error)
	dbQueryRow(string, ...interface{}) *sql.Row
}

type dbConn struct {
	db *sql.DB
}

func (d *dbConn) dbExec(qs string, args ...interface{}) error {
	_, err := d.db.Exec(qs, args...)
	return err
}

func (d *dbConn) dbQuery(qs string, args ...interface{}) (*sql.Rows, error) {
	return d.db.Query(qs, args...)
}

func (d *dbConn) dbQueryRow(qs string, args ...interface{}) *sql.Row {
	return d.db.QueryRow(qs, args...)
}

type dbTx struct {
	tx *sql.Tx
}

func (d *dbTx) Commit() error {
	return d.tx.Commit()
}

func (d *dbTx) Rollback() error {
	return d.tx.Rollback()
}

func (d *dbTx) dbExec(qs string, args ...interface{}) error {
	_, err := d.tx.Exec(qs, args...)
	return err
}

func (d *dbTx) dbQuery(qs string, args ...interface{}) (*sql.Rows, error) {
	return d.tx.Query(qs, args...)
}

func (d *dbTx) dbQueryRow(qs string, args ...interface{}) *sql.Row {
	return d.tx.QueryRow(qs, args...)
}

func dbNewTx() (ret dbTx, err error) {
	ret.tx, err = db.db.Begin()
	return ret, err
}

func dbInit(c config) error {
	var err error
	log.logf("initializing database connection to %v", c.Database.Host)
	connstr := fmt.Sprintf("dbname=%v host=%v user=%v password=%v", c.Database.Database,
		c.Database.Host, c.Database.Username, c.Database.Password)
	db.db, err = sql.Open("postgres", connstr)
	return err
}

// Given a raw indicator, locate the asset associated with that indicator in the
// database. If the asset is not found, a new one will be added and this asset
// will be returned.
func dbLocateAssetFromIndicator(indicator slib.RawIndicator) (ret slib.Asset, err error) {
	var aid int
	err = db.dbQueryRow(`SELECT assetid FROM asset
		WHERE assettype = $1 AND name = $2 AND zone = $3`, indicator.Type,
		indicator.Name, indicator.Zone).Scan(&aid)
	if err == nil {
		// Asset was found
		return dbGetAssetID(aid)
	}
	if err != sql.ErrNoRows {
		return
	}
	// Otherwise, add the new asset and return it
	err = db.dbQueryRow(`INSERT INTO asset
		(assettype, name, zone, description, lastindicator)
		VALUES ($1, $2, $3, $4, $5) RETURNING assetid`,
		indicator.Type, indicator.Name, indicator.Zone,
		indicator.Description, indicator.Timestamp).Scan(&aid)
	if err != nil {
		return
	}
	return dbGetAssetID(aid)
}

// Get an asset from the database by ID
func dbGetAssetID(aid int) (ret slib.Asset, err error) {
	var desc sql.NullString
	err = db.dbQueryRow(`SELECT assetid, assettype,
		name, zone, description, lastindicator
		FROM asset WHERE assetid = $1`, aid).Scan(&ret.ID,
		&ret.Type, &ret.Name, &ret.Zone, &desc,
		&ret.LastIndicator)
	if err != nil {
		return
	}
	if desc.Valid {
		ret.Description = desc.String
	}
	return
}

// Add an owner to the database
func dbAddOwner(dbi dbInterface, own slib.Owner) error {
	return dbi.dbExec(`INSERT INTO assetowners
		(team, operator)
		SELECT $1, $2
		WHERE NOT EXISTS (
			SELECT 1 FROM assetowners
			WHERE team = $3 AND operator = $4
		)`, own.Team, own.Operator, own.Team, own.Operator)
}

// Return all owners from the database
func dbGetOwners(dbi dbInterface) (ret []slib.Owner, err error) {
	rows, err := dbi.dbQuery(`SELECT ownerid, team, operator
		FROM assetowners`)
	if err != nil {
		return
	}
	for rows.Next() {
		var own slib.Owner
		err = rows.Scan(&own.ID, &own.Team, &own.Operator)
		if err != nil {
			rows.Close()
			return ret, err
		}
		ret = append(ret, own)
	}
	if err = rows.Err(); err != nil {
		return ret, err
	}
	return
}

// Remove owner own from the database
func dbRemoveOwner(dbi dbInterface, own slib.Owner) error {
	err := dbi.dbExec(`UPDATE asset
		SET ownerid = NULL
		WHERE ownerid = $1`,
		own.ID)
	if err != nil {
		return err
	}
	return dbi.dbExec(`DELETE FROM assetowners
		WHERE ownerid = $1`, own.ID)
}

// Apply ownership configuration for "hostname" type assets. Any asset entries in
// the database that match the regular expression in hostmatch will be linked to the
// indicated team/operator. If triage is not "", this triageoverride will also be
// applied to matching assets.
//
// Ensure the owner exists in the database prior to attempting to link.
func dbHostOwnership(dbi dbInterface, hostmatch string, team string, operator string, triage string) error {
	var tover sql.NullString
	if triage != "" {
		tover.String = triage
		tover.Valid = true
	}
	err := dbi.dbExec(`UPDATE asset
		SET ownerid = (
			SELECT ownerid FROM assetowners
			WHERE team = $1 AND operator = $2
		), triageoverride = $3 WHERE assettype = 'hostname' AND
		name ~* $4`, team, operator, tover, hostmatch)
	if err != nil {
		return err
	}
	return nil
}

// Add an asset group to the database
func dbAddAssetGroup(dbi dbInterface, grp slib.AssetGroup) error {
	return dbi.dbExec(`INSERT INTO assetgroup
		(name)
		SELECT $1
		WHERE NOT EXISTS (
			SELECT assetgroupid FROM assetgroup WHERE name = $2
		)`, grp.Name, grp.Name)
}

// Remove an asset group from the database
func dbRemoveAssetGroup(dbi dbInterface, grp slib.AssetGroup) error {
	err := dbi.dbExec(`UPDATE asset
		SET assetgroupid = NULL
		WHERE assetgroupid = $1`,
		grp.ID)
	if err != nil {
		return err
	}
	err = dbi.dbExec(`DELETE FROM rra_assetgroup
		WHERE assetgroupid = $1`,
		grp.ID)
	if err != nil {
		return err
	}
	return dbi.dbExec(`DELETE FROM assetgroup
		WHERE assetgroupid = $1`, grp.ID)
}

// Return all asset groups present in the database
func dbGetAssetGroups(dbi dbInterface) (ret []slib.AssetGroup, err error) {
	rows, err := dbi.dbQuery(`SELECT assetgroupid, name
		FROM assetgroup`)
	if err != nil {
		return
	}
	for rows.Next() {
		var grp slib.AssetGroup
		err = rows.Scan(&grp.ID, &grp.Name)
		if err != nil {
			rows.Close()
			return ret, err
		}
		ret = append(ret, grp)
	}
	if err = rows.Err(); err != nil {
		return ret, err
	}
	return
}

// For the asset group named grp, unlink this group from any "hostname" type assets
// which it is currently associated with.
func dbPurgeAssetGroupHosts(dbi dbInterface, grp string) error {
	return dbi.dbExec(`UPDATE asset
		SET assetgroupid = NULL
		WHERE assetgroupid = (
			SELECT assetgroupid FROM assetgroup
			WHERE name = $1
		) AND assettype = 'hostname'`, grp)
}

// For "hostname" type assets matching the regular expression hostmatch, link these
// assets to the asset group named destgrp.
func dbHostLinkAssetGroup(dbi dbInterface, hostmatch string, destgrp string) error {
	return dbi.dbExec(`UPDATE asset
		SET assetgroupid = (
			SELECT assetgroupid FROM assetgroup
			WHERE name = $1
		) WHERE assettype = 'hostname' AND
		name ~* $2`,
		destgrp, hostmatch)
}

// Link any RRAs that match the regular expression destrra to the asset group
// named grp in the database.
func dbAssetGroupLinkRRA(dbi dbInterface, grp string, destrra string) error {
	var rraids []int
	rows, err := dbi.dbQuery(`SELECT rraid FROM rra
		WHERE service ~* $1`, destrra)
	if err != nil {
		return err
	}
	for rows.Next() {
		var rraid int
		err = rows.Scan(&rraid)
		if err != nil {
			rows.Close()
			return err
		}
		rraids = append(rraids, rraid)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	for _, x := range rraids {
		err = dbi.dbExec(`INSERT INTO rra_assetgroup
			(rraid, assetgroupid)
			SELECT $1, (
				SELECT assetgroupid FROM assetgroup
				WHERE name = $2
			) WHERE NOT EXISTS (
				SELECT 1 FROM rra_assetgroup
				WHERE rraid = $3 AND
				assetgroupid = (
					SELECT assetgroupid FROM assetgroup
					WHERE name = $4
				)
			)`, x, grp, x, grp)
		if err != nil {
			return err
		}
	}
	return nil
}

// Remove any linkage from RRAs in the dataabase to the asset group named grp.
func dbPurgeAssetGroupRRAs(dbi dbInterface, grp string) error {
	return dbi.dbExec(`DELETE FROM rra_assetgroup
		WHERE assetgroupid = (
			SELECT assetgroupid FROM assetgroup
			WHERE name = $1
		)`, grp)
}
