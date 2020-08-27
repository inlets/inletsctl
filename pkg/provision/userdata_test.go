// Copyright (c) Inlets Author(s) 2020. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package provision

import (
	"io/ioutil"
	"testing"
)

func Test_makeUserdata_InletsOSS(t *testing.T) {
	userData := MakeExitServerUserdata(8080, "auth", "2.7.4", "0.7.0", false)

	wantUserdata := `#!/bin/bash
export AUTHTOKEN="auth"
export CONTROLPORT="8080"

curl -SLsf https://github.com/inlets/inlets/releases/download/2.7.4/inlets > /tmp/inlets && \
chmod +x /tmp/inlets  && \
mv /tmp/inlets /usr/local/bin/inlets

curl -sLO https://raw.githubusercontent.com/inlets/inlets/master/hack/inlets-operator.service && \
mv inlets-operator.service /etc/systemd/system/inlets.service && \
echo "AUTHTOKEN=$AUTHTOKEN" > /etc/default/inlets && \
echo "CONTROLPORT=$CONTROLPORT" >> /etc/default/inlets && \
systemctl start inlets && \
systemctl enable inlets
`

	ioutil.WriteFile("/tmp/oss", []byte(userData), 0600)

	if userData != wantUserdata {
		t.Errorf("want:\n%s\nbut got:\n%s", wantUserdata, userData)
	}
}

func Test_makeUserdata_InletsPro(t *testing.T) {
	userData := MakeExitServerUserdata(8080, "auth", "2.7.4", "0.7.0", true)

	wantUserdata := `#!/bin/bash
export AUTHTOKEN="auth"
export IP=$(curl -sfSL https://checkip.amazonaws.com)

curl -SLsf https://github.com/inlets/inlets-pro/releases/download/0.7.0/inlets-pro > /tmp/inlets-pro && \
  chmod +x /tmp/inlets-pro  && \
  mv /tmp/inlets-pro /usr/local/bin/inlets-pro

curl -sLO https://raw.githubusercontent.com/inlets/inlets-pro/master/artifacts/inlets-pro.service  && \
  mv inlets-pro.service /etc/systemd/system/inlets-pro.service && \
  echo "AUTHTOKEN=$AUTHTOKEN" >> /etc/default/inlets-pro && \
  echo "IP=$IP" >> /etc/default/inlets-pro && \
  systemctl start inlets-pro && \
  systemctl enable inlets-pro
`

	// ioutil.WriteFile("/tmp/pro", []byte(userData), 0600)
	if userData != wantUserdata {
		t.Errorf("want: %s, but got: %s", wantUserdata, userData)
	}
}
