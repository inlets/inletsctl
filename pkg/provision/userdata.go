package provision

import "fmt"

// MakeExitServerUserdata makes a user-data script in bash to setup inlets
// using the OSS or PRO version with a systemd service
func MakeExitServerUserdata(ossControlPort int, authToken, ossVersion, proVersion string, pro bool) string {
	if pro {
		return `#!/bin/bash
export AUTHTOKEN="` + authToken + `"
export IP=$(curl -sfSL https://checkip.amazonaws.com)

curl -SLsf https://github.com/inlets/inlets-pro/releases/download/` + proVersion + `/inlets-pro > /tmp/inlets-pro && \
  chmod +x /tmp/inlets-pro  && \
  mv /tmp/inlets-pro /usr/local/bin/inlets-pro

curl -sLO https://raw.githubusercontent.com/inlets/inlets-pro/master/artifacts/inlets-pro.service  && \
  mv inlets-pro.service /etc/systemd/system/inlets-pro.service && \
  echo "AUTHTOKEN=$AUTHTOKEN" >> /etc/default/inlets-pro && \
  echo "IP=$IP" >> /etc/default/inlets-pro && \
  systemctl start inlets-pro && \
  systemctl enable inlets-pro
`
	}

	controlPort := fmt.Sprintf("%d", ossControlPort)

	return `#!/bin/bash
export AUTHTOKEN="` + authToken + `"
export CONTROLPORT="` + controlPort + `"

curl -SLsf https://github.com/inlets/inlets/releases/download/` + ossVersion + `/inlets > /tmp/inlets && \
chmod +x /tmp/inlets  && \
mv /tmp/inlets /usr/local/bin/inlets

curl -sLO https://raw.githubusercontent.com/inlets/inlets/master/hack/inlets-operator.service && \
mv inlets-operator.service /etc/systemd/system/inlets.service && \
echo "AUTHTOKEN=$AUTHTOKEN" > /etc/default/inlets && \
echo "CONTROLPORT=$CONTROLPORT" >> /etc/default/inlets && \
systemctl start inlets && \
systemctl enable inlets
`
}
