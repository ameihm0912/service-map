#!/usr/bin/env python
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.
# Copyright (c) 2015 Mozilla Corporation
# Author: ameihm@mozilla.com

import requests
import uuid
import json

class SLIBException(Exception):
    def __init__(self, m):
        self._message = m

    def __str__(self):
        return self._message

class Search(object):
    def __init__(self, url, verify=True):
        self._url = url
        self._searches = {}
        self._verify = verify

    def add_host(self, hostname):
        self._searches[str(uuid.uuid4())] = {'host': hostname}

    def result_host(self, hostname):
        for x in self._searches:
            res = self._searches[x]
            if 'host' not in res:
                continue
            if res['host'] != hostname:
                continue
            return res['result']['service']
        return None

    def execute(self):
        searchlist = []
        if len(self._searches) == 0:
            return
        for x in self._searches:
            n = self._searches[x]
            n['identifier'] = x
            searchlist.append(n)
        s = {'search': searchlist}
        u = self._url + '/search'
        payload = {'params': json.dumps(s)}
        r = requests.post(u, data=payload, verify=self._verify)
        if r.status_code != requests.codes.ok:
            err = 'Search error: response {}, "{}"'.format(r.status_code, r.text.strip())
            raise SLIBException(err)
        resp = json.loads(r.text)

        payload = {'id': resp['id']}
        u = self._url + '/search/results/id'
        r = requests.get(u, params=payload, verify=self._verify)
        if r.status_code != requests.codes.ok:
            err = 'Search results error: response {}, "{}"'.format(r.status_code, r.text.strip())
            raise SLIBException(err)
        resp = json.loads(r.text)

        for x in resp['results']:
            self._searches[x['identifier']]['result'] = x
