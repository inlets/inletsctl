// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"testing"
)

func Test_makeUserdata_InletsOSS(t *testing.T) {
	userData := makeUserdata("auth", inletsControlPort, "")

	wantUserdata := `#!/bin/bash
export AUTHTOKEN="auth"
export CONTROLPORT="8080"
curl -sLS https://get.inlets.dev | sh

curl -sLO https://raw.githubusercontent.com/inlets/inlets/master/hack/inlets-operator.service  && \
  mv inlets-operator.service /etc/systemd/system/inlets.service && \
  echo "AUTHTOKEN=$AUTHTOKEN" > /etc/default/inlets && \
  echo "CONTROLPORT=$CONTROLPORT" >> /etc/default/inlets && \
  systemctl start inlets && \
  systemctl enable inlets`

	if userData != wantUserdata {
		t.Errorf("want:\n%s\nbut got:\n%s", wantUserdata, userData)
	}
}

func Test_makeUserdata_InletsPro(t *testing.T) {
	userData := makeUserdata("auth", inletsProControlPort, "localhost")

	wantUserdata := `#!/bin/bash
export AUTHTOKEN="auth"
export REMOTETCP="localhost"
export IP=$(curl -sfSL https://ifconfig.co)

curl -SLsf https://github.com/inlets/inlets-pro/releases/download/0.4.3/inlets-pro > /tmp/inlets-pro && \
  chmod +x /tmp/inlets-pro  && \
  mv /tmp/inlets-pro /usr/local/bin/inlets-pro

curl -sLO https://raw.githubusercontent.com/inlets/inlets/master/hack/inlets-pro.service  && \
  mv inlets-pro.service /etc/systemd/system/inlets-pro.service && \
  echo "AUTHTOKEN=$AUTHTOKEN" >> /etc/default/inlets-pro && \
  echo "REMOTETCP=$REMOTETCP" >> /etc/default/inlets-pro && \
  echo "IP=$IP" >> /etc/default/inlets-pro && \
  systemctl start inlets-pro && \
  systemctl enable inlets-pro`

	if userData != wantUserdata {
		t.Errorf("want: %s, but got: %s", wantUserdata, userData)
	}
}
