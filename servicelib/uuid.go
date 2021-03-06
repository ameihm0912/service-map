// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package servicelib

import (
	"github.com/pborman/uuid"
)

// NewUUID gneerates and returns a new UUID value
func NewUUID() string {
	return uuid.New()
}
