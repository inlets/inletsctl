package provision

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/inlets/inletsctl/mock"
	"github.com/linode/linodego"
	"net"
	"testing"
)

func Test_Linode_Provision(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mock.NewMockLinodeInterface(mockCtrl)
	provisioner := &LinodeProvisioner{
		client: mockClient,
	}
	host := BasicHost{
		Region:     "us-east",
		Plan:       "g6-nanode-1",
		OS:         "linode/ubuntu16.04lts",
		Name:       "testvm",
		UserData:   "user-data",
		Additional: nil,
	}
	stackscriptOption := linodego.StackscriptCreateOptions{
		IsPublic: false, Images: []string{host.OS}, Script: host.UserData, Label: host.Name,
	}
	returnedStackscript := &linodego.Stackscript{ID: 10}

	expectedInstanceOptions := linodego.InstanceCreateOptions{
		Label: "inlets-" + host.Name, StackScriptID: returnedStackscript.ID,
		Image: host.OS, Region: host.Region, Type: host.Plan, RootPass: "testpass",
	}
	returnedInstance := &linodego.Instance{
		ID:     42,
		Status: linodego.InstanceBooting,
	}

	mockClient.EXPECT().CreateStackscript(gomock.Eq(stackscriptOption)).Return(returnedStackscript, nil).Times(1)
	mockClient.EXPECT().CreateInstance(gomock.Any()).Return(returnedInstance, nil).Times(1).
		Do(func(instanceOptions linodego.InstanceCreateOptions) {
			if instanceOptions.Label != expectedInstanceOptions.Label {
				t.Fail()
			}
			if instanceOptions.StackScriptID != expectedInstanceOptions.StackScriptID {
				t.Fail()
			}
			if instanceOptions.Image != expectedInstanceOptions.Image {
				t.Fail()
			}
			if instanceOptions.Region != expectedInstanceOptions.Region {
				t.Fail()
			}
			if instanceOptions.Type != expectedInstanceOptions.Type {
				t.Fail()
			}
		})
	provisionedHost, _ := provisioner.Provision(host)
	if provisionedHost.ID != fmt.Sprintf("%d", returnedInstance.ID) {
		t.Errorf("provisionedHost.ID want: %v, but got: %v", returnedInstance.ID, provisionedHost.ID)
	}
	if provisionedHost.Status != string(returnedInstance.Status) {
		t.Errorf("provisionedHost.Status want: %v, but got: %v", returnedInstance.Status, provisionedHost.Status)
	}
}

func Test_Linode_StatusBooting(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mock.NewMockLinodeInterface(mockCtrl)
	provisioner := &LinodeProvisioner{
		client: mockClient,
	}

	instanceId := 42
	returnedInstance := &linodego.Instance{
		ID:     42,
		Status: linodego.InstanceBooting,
	}
	expectedReturn := &ProvisionedHost{
		IP:     "",
		ID:     fmt.Sprintf("%d", instanceId),
		Status: string(returnedInstance.Status),
	}

	mockClient.EXPECT().GetInstance(gomock.Eq(instanceId)).Return(returnedInstance, nil).Times(1)
	provisionedHost, _ := provisioner.Status(fmt.Sprintf("%d", instanceId))
	if *expectedReturn != *provisionedHost {
		t.Errorf("provisionedHost want: %v, but got: %v", expectedReturn, provisionedHost)
	}
}

func Test_Linode_StatusActive(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mock.NewMockLinodeInterface(mockCtrl)
	provisioner := &LinodeProvisioner{
		client:        mockClient,
		stackscriptID: 10,
	}

	instanceId := 42
	ip := net.IPv4(127, 0, 0, 1)
	returnedInstance := &linodego.Instance{
		ID:     42,
		Status: linodego.InstanceRunning,
		IPv4:   []*net.IP{&ip},
	}
	expectedReturn := &ProvisionedHost{
		IP:     ip.String(),
		ID:     fmt.Sprintf("%d", instanceId),
		Status: ActiveStatus,
	}

	mockClient.EXPECT().GetInstance(gomock.Eq(instanceId)).Return(returnedInstance, nil).Times(1)
	mockClient.EXPECT().DeleteStackscript(gomock.Eq(provisioner.stackscriptID)).Return(nil).Times(1)
	provisionedHost, _ := provisioner.Status(fmt.Sprintf("%d", instanceId))
	if *expectedReturn != *provisionedHost {
		t.Errorf("provisionedHost want: %v, but got: %v", expectedReturn, provisionedHost)
	}
}

func Test_Linode_Delete(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockClient := mock.NewMockLinodeInterface(mockCtrl)
	provisioner := &LinodeProvisioner{
		client: mockClient,
	}
	instanceId := 42
	request := HostDeleteRequest{
		ID: "42",
	}

	mockClient.EXPECT().DeleteInstance(gomock.Eq(instanceId)).Return(nil).Times(1)
	_ = provisioner.Delete(request)
}
