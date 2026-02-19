// Copyright (c) Inlets Author(s) 2023. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"os"
	"testing"
)

func Test_MakeHTTPSUserdata_OneDomain(t *testing.T) {
	got := MakeHTTPSUserdata("token", "0.9.40", "prod", []string{"example.com"})

	os.WriteFile("/tmp/t.txt", []byte(got), 0600)
	want := `#!/bin/bash
export AUTHTOKEN="token"
export IP=$(curl -sfSL https://checkip.amazonaws.com)

curl -SLsf https://github.com/inlets/inlets-pro/releases/download/0.9.40/inlets-pro -o /tmp/inlets-pro && \
  chmod +x /tmp/inlets-pro  && \
  mv /tmp/inlets-pro /usr/local/bin/inlets-pro

curl -SLsf https://github.com/inlets/inlets-pro/releases/download/0.9.40/inlets-pro-http.service -o inlets-pro.service && \
  mv inlets-pro.service /etc/systemd/system/inlets-pro.service && \
  echo "AUTHTOKEN=$AUTHTOKEN" >> /etc/default/inlets-pro && \
  echo "IP=$IP" >> /etc/default/inlets-pro && \
  echo "DOMAINS=--letsencrypt-domain=example.com" >> /etc/default/inlets-pro && \
  echo "ISSUER=--letsencrypt-issuer=prod" >> /etc/default/inlets-pro && \
  systemctl daemon-reload && \
  systemctl start inlets-pro && \
  systemctl enable inlets-pro
`
	if want != got {
		t.Fatalf("want\n%s\nbut got\n%s\n", want, got)
	}
}

func Test_MakeHTTPSUserdata_TwoDomains(t *testing.T) {
	got := MakeHTTPSUserdata("token", "0.9.40", "prod",
		[]string{"a.example.com", "b.example.com"})

	os.WriteFile("/tmp/t.txt", []byte(got), 0600)
	want := `#!/bin/bash
export AUTHTOKEN="token"
export IP=$(curl -sfSL https://checkip.amazonaws.com)

curl -SLsf https://github.com/inlets/inlets-pro/releases/download/0.9.40/inlets-pro -o /tmp/inlets-pro && \
  chmod +x /tmp/inlets-pro  && \
  mv /tmp/inlets-pro /usr/local/bin/inlets-pro

curl -SLsf https://github.com/inlets/inlets-pro/releases/download/0.9.40/inlets-pro-http.service -o inlets-pro.service && \
  mv inlets-pro.service /etc/systemd/system/inlets-pro.service && \
  echo "AUTHTOKEN=$AUTHTOKEN" >> /etc/default/inlets-pro && \
  echo "IP=$IP" >> /etc/default/inlets-pro && \
  echo "DOMAINS=--letsencrypt-domain=a.example.com --letsencrypt-domain=b.example.com" >> /etc/default/inlets-pro && \
  echo "ISSUER=--letsencrypt-issuer=prod" >> /etc/default/inlets-pro && \
  systemctl daemon-reload && \
  systemctl start inlets-pro && \
  systemctl enable inlets-pro
`
	if want != got {
		t.Fatalf("want\n%s\nbut got\n%s\n", want, got)
	}
}
