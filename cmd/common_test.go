// Copyright (c) Inlets Author(s) 2023. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import "testing"

func Test_subscriptionIDFromAzureAuth(t *testing.T) {
	t.Parallel()

	auth := `{"subscriptionId":"sub-from-file","clientId":"id","clientSecret":"secret","tenantId":"tenant"}`

	if got := subscriptionIDFromAzureAuth("", auth); got != "sub-from-file" {
		t.Fatalf("want sub-from-file, got %q", got)
	}

	if got := subscriptionIDFromAzureAuth("sub-from-flag", auth); got != "sub-from-flag" {
		t.Fatalf("want sub-from-flag, got %q", got)
	}

	if got := subscriptionIDFromAzureAuth("", "not-json"); got != "" {
		t.Fatalf("want empty string, got %q", got)
	}
}

func Test_sampleDeleteCommand(t *testing.T) {
	t.Parallel()

	want := `inletsctl delete --provider digitalocean --id "123"`
	if got := sampleDeleteCommand("digitalocean", "123", ""); got != want {
		t.Fatalf("want %q, got %q", want, got)
	}

	want = `inletsctl delete --provider azure --id "group|deployment" --subscription-id "00000000-0000-0000-0000-000000000000"`
	if got := sampleDeleteCommand("azure", "group|deployment", "00000000-0000-0000-0000-000000000000"); got != want {
		t.Fatalf("want %q, got %q", want, got)
	}

	want = `inletsctl delete --provider azure --id "group|deployment"`
	if got := sampleDeleteCommand("azure", "group|deployment", ""); got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}
