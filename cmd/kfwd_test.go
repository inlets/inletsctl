// Copyright (c) Inlets Author(s) 2023. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"strings"
	"testing"
)

func Test_MakeTCPDeployment_UsesDefaultInletsProVersion(t *testing.T) {
	got := makeTCPDeployment("192.168.0.10", "4222", "nats", "default", "token", "license")

	for _, want := range []string{
		"image: ghcr.io/inlets/inlets-pro:" + inletsProDefaultVersion,
		"- \"tcp\"",
		"- \"client\"",
		"- \"--url=wss://192.168.0.10:8123\"",
		"- \"--upstream=nats\"",
		"- \"--ports=4222\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected generated TCP deployment to contain %q, got:\n%s", want, got)
		}
	}
}

func Test_MakeHTTPDeployment_UsesDefaultInletsProVersion(t *testing.T) {
	got := makeHTTPDeployment("192.168.0.10", "8080", "web", "default", "token", "license")

	for _, want := range []string{
		"image: ghcr.io/inlets/inlets-pro:" + inletsProDefaultVersion,
		"- \"http\"",
		"- \"client\"",
		"- \"--url=wss://192.168.0.10:8123\"",
		"- \"--upstream=http://web:8080\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected generated HTTP deployment to contain %q, got:\n%s", want, got)
		}
	}
}
