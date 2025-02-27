/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAzureMachineSpec_SetDefaultSSHPublicKey(t *testing.T) {
	g := NewWithT(t)

	type test struct {
		machine *AzureMachine
	}

	existingPublicKey := "testpublickey"
	publicKeyExistTest := test{machine: createMachineWithSSHPublicKey(existingPublicKey)}
	publicKeyNotExistTest := test{machine: createMachineWithSSHPublicKey("")}

	err := publicKeyExistTest.machine.Spec.SetDefaultSSHPublicKey()
	g.Expect(err).To(BeNil())
	g.Expect(publicKeyExistTest.machine.Spec.SSHPublicKey).To(Equal(existingPublicKey))

	err = publicKeyNotExistTest.machine.Spec.SetDefaultSSHPublicKey()
	g.Expect(err).To(BeNil())
	g.Expect(publicKeyNotExistTest.machine.Spec.SSHPublicKey).To(Not(BeEmpty()))
}

func TestAzureMachineSpec_SetIdentityDefaults(t *testing.T) {
	g := NewWithT(t)

	type test struct {
		machine *AzureMachine
	}

	fakeSubscriptionID := uuid.New().String()
	fakeClusterName := "testcluster"
	fakeRoleDefinitionID := "testroledefinitionid"
	fakeScope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", fakeSubscriptionID, fakeClusterName)
	existingRoleAssignmentName := "42862306-e485-4319-9bf0-35dbc6f6fe9c"
	roleAssignmentExistTest := test{machine: &AzureMachine{Spec: AzureMachineSpec{
		Identity: VMIdentitySystemAssigned,
		SystemAssignedIdentityRole: &SystemAssignedIdentityRole{
			Name: existingRoleAssignmentName,
		},
	}}}
	notSystemAssignedTest := test{machine: &AzureMachine{Spec: AzureMachineSpec{
		Identity:                   VMIdentityUserAssigned,
		SystemAssignedIdentityRole: &SystemAssignedIdentityRole{},
	}}}
	systemAssignedIdentityRoleExistTest := test{machine: &AzureMachine{Spec: AzureMachineSpec{
		Identity: VMIdentitySystemAssigned,
		SystemAssignedIdentityRole: &SystemAssignedIdentityRole{
			Scope:        fakeScope,
			DefinitionID: fakeRoleDefinitionID,
		},
	}}}

	emptyTest := test{machine: &AzureMachine{Spec: AzureMachineSpec{
		Identity:                   VMIdentitySystemAssigned,
		SystemAssignedIdentityRole: &SystemAssignedIdentityRole{},
	}}}

	roleAssignmentExistTest.machine.Spec.SetIdentityDefaults(fakeSubscriptionID)
	g.Expect(roleAssignmentExistTest.machine.Spec.SystemAssignedIdentityRole.Name).To(Equal(existingRoleAssignmentName))

	notSystemAssignedTest.machine.Spec.SetIdentityDefaults(fakeSubscriptionID)
	g.Expect(notSystemAssignedTest.machine.Spec.SystemAssignedIdentityRole.Name).To(BeEmpty())

	systemAssignedIdentityRoleExistTest.machine.Spec.SetIdentityDefaults(fakeSubscriptionID)
	g.Expect(systemAssignedIdentityRoleExistTest.machine.Spec.SystemAssignedIdentityRole.Scope).To(Equal(fakeScope))
	g.Expect(systemAssignedIdentityRoleExistTest.machine.Spec.SystemAssignedIdentityRole.DefinitionID).To(Equal(fakeRoleDefinitionID))

	emptyTest.machine.Spec.SetIdentityDefaults(fakeSubscriptionID)
	g.Expect(emptyTest.machine.Spec.SystemAssignedIdentityRole.Name).To(Not(BeEmpty()))
	_, err := uuid.Parse(emptyTest.machine.Spec.SystemAssignedIdentityRole.Name)
	g.Expect(err).To(Not(HaveOccurred()))
	g.Expect(emptyTest.machine.Spec.SystemAssignedIdentityRole.Scope).To(Equal(fmt.Sprintf("/subscriptions/%s/", fakeSubscriptionID)))
	g.Expect(emptyTest.machine.Spec.SystemAssignedIdentityRole.DefinitionID).To(Equal(fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s", fakeSubscriptionID, ContributorRoleID)))
}

func TestAzureMachineSpec_SetDataDisksDefaults(t *testing.T) {
	cases := []struct {
		name   string
		disks  []DataDisk
		output []DataDisk
	}{
		{
			name:   "no disks",
			disks:  []DataDisk{},
			output: []DataDisk{},
		},
		{
			name: "no LUNs specified",
			disks: []DataDisk{
				{
					NameSuffix:  "testdisk1",
					DiskSizeGB:  30,
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk2",
					DiskSizeGB:  30,
					CachingType: "ReadWrite",
				},
			},
			output: []DataDisk{
				{
					NameSuffix:  "testdisk1",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(0),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk2",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(1),
					CachingType: "ReadWrite",
				},
			},
		},
		{
			name: "All LUNs specified",
			disks: []DataDisk{
				{
					NameSuffix:  "testdisk1",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(5),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk2",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(3),
					CachingType: "ReadWrite",
				},
			},
			output: []DataDisk{
				{
					NameSuffix:  "testdisk1",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(5),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk2",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(3),
					CachingType: "ReadWrite",
				},
			},
		},
		{
			name: "Some LUNs missing",
			disks: []DataDisk{
				{
					NameSuffix:  "testdisk1",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(0),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk2",
					DiskSizeGB:  30,
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk3",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(1),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk4",
					DiskSizeGB:  30,
					CachingType: "ReadWrite",
				},
			},
			output: []DataDisk{
				{
					NameSuffix:  "testdisk1",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(0),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk2",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(2),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk3",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(1),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk4",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(3),
					CachingType: "ReadWrite",
				},
			},
		},
		{
			name: "CachingType unspecified",
			disks: []DataDisk{
				{
					NameSuffix: "testdisk1",
					DiskSizeGB: 30,
					Lun:        pointer.Int32(0),
				},
				{
					NameSuffix: "testdisk2",
					DiskSizeGB: 30,
					Lun:        pointer.Int32(2),
				},
				{
					NameSuffix: "testdisk3",
					DiskSizeGB: 30,
					ManagedDisk: &ManagedDiskParameters{
						StorageAccountType: "UltraSSD_LRS",
					},
					Lun: pointer.Int32(3),
				},
			},
			output: []DataDisk{
				{
					NameSuffix:  "testdisk1",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(0),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix:  "testdisk2",
					DiskSizeGB:  30,
					Lun:         pointer.Int32(2),
					CachingType: "ReadWrite",
				},
				{
					NameSuffix: "testdisk3",
					DiskSizeGB: 30,
					Lun:        pointer.Int32(3),
					ManagedDisk: &ManagedDiskParameters{
						StorageAccountType: "UltraSSD_LRS",
					},
					CachingType: "None",
				},
			},
		},
	}

	for _, c := range cases {
		tc := c
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			machine := hardcodedAzureMachineWithSSHKey(generateSSHPublicKey(true))
			machine.Spec.DataDisks = tc.disks
			machine.Spec.SetDataDisksDefaults()
			if !reflect.DeepEqual(machine.Spec.DataDisks, tc.output) {
				expected, _ := json.MarshalIndent(tc.output, "", "\t")
				actual, _ := json.MarshalIndent(machine.Spec.DataDisks, "", "\t")
				t.Errorf("Expected %s, got %s", string(expected), string(actual))
			}
		})
	}
}

func TestAzureMachineSpec_SetNetworkInterfacesDefaults(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name    string
		machine *AzureMachine
		want    *AzureMachine
	}{
		{
			name: "defaulting webhook updates machine with deprecated subnetName field",
			machine: &AzureMachine{
				Spec: AzureMachineSpec{
					SubnetName: "test-subnet",
				},
			},
			want: &AzureMachine{
				Spec: AzureMachineSpec{
					SubnetName: "",
					NetworkInterfaces: []NetworkInterface{
						{
							SubnetName:       "test-subnet",
							PrivateIPConfigs: 1,
						},
					},
				},
			},
		},
		{
			name: "defaulting webhook updates machine with deprecated subnetName field and empty NetworkInterfaces slice",
			machine: &AzureMachine{
				Spec: AzureMachineSpec{
					SubnetName:        "test-subnet",
					NetworkInterfaces: []NetworkInterface{},
				},
			},
			want: &AzureMachine{
				Spec: AzureMachineSpec{
					SubnetName: "",
					NetworkInterfaces: []NetworkInterface{
						{
							SubnetName:       "test-subnet",
							PrivateIPConfigs: 1,
						},
					},
				},
			},
		},
		{
			name: "defaulting webhook updates machine with deprecated acceleratedNetworking field",
			machine: &AzureMachine{
				Spec: AzureMachineSpec{
					SubnetName:            "test-subnet",
					AcceleratedNetworking: pointer.Bool(true),
				},
			},
			want: &AzureMachine{
				Spec: AzureMachineSpec{
					SubnetName:            "",
					AcceleratedNetworking: nil,
					NetworkInterfaces: []NetworkInterface{
						{
							SubnetName:            "test-subnet",
							PrivateIPConfigs:      1,
							AcceleratedNetworking: pointer.Bool(true),
						},
					},
				},
			},
		},
		{
			name: "defaulting webhook does nothing if both new and deprecated subnetName fields are set",
			machine: &AzureMachine{
				Spec: AzureMachineSpec{
					SubnetName: "test-subnet",
					NetworkInterfaces: []NetworkInterface{{
						SubnetName: "test-subnet",
					}},
				},
			},
			want: &AzureMachine{
				Spec: AzureMachineSpec{
					SubnetName:            "test-subnet",
					AcceleratedNetworking: nil,
					NetworkInterfaces: []NetworkInterface{
						{
							SubnetName: "test-subnet",
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.machine.Spec.SetNetworkInterfacesDefaults()
			g.Expect(tc.machine).To(Equal(tc.want))
		})
	}
}

func TestAzureMachineSpec_GetSubscriptionID(t *testing.T) {
	g := NewWithT(t)

	tests := []struct {
		name        string
		maxAttempts int
		want        string
		wantErr     bool
	}{
		{
			name:        "subscription ID is returned",
			maxAttempts: 1,
			want:        "test-subscription-id",
			wantErr:     false,
		},
		{
			name:        "subscription ID is returned after 2 attempts",
			maxAttempts: 2,
			want:        "test-subscription-id",
			wantErr:     false,
		},
		{
			name:        "subscription ID is not returned after 5 attempts",
			maxAttempts: 5,
			want:        "",
			wantErr:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := mockClient{ReturnError: tc.wantErr}
			result, err := GetSubscriptionID(client, "test-cluster", "default", tc.maxAttempts)
			if tc.wantErr {
				g.Expect(err).To(HaveOccurred())
			} else {
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(result).To(Equal(tc.want))
			}
		})
	}
}

type mockClient struct {
	client.Client
	ReturnError bool
}

func (m mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if m.ReturnError {
		return errors.New("AzureCluster not found: failed to find owner cluster for test-cluster")
	}
	ac := &AzureCluster{}
	ac.Spec.SubscriptionID = "test-subscription-id"
	obj.(*AzureCluster).Spec.SubscriptionID = ac.Spec.SubscriptionID

	return nil
}

func createMachineWithSSHPublicKey(sshPublicKey string) *AzureMachine {
	machine := hardcodedAzureMachineWithSSHKey(sshPublicKey)
	return machine
}

func createMachineWithUserAssignedIdentities(identitiesList []UserAssignedIdentity) *AzureMachine {
	machine := hardcodedAzureMachineWithSSHKey(generateSSHPublicKey(true))
	machine.Spec.Identity = VMIdentityUserAssigned
	machine.Spec.UserAssignedIdentities = identitiesList
	return machine
}

func hardcodedAzureMachineWithSSHKey(sshPublicKey string) *AzureMachine {
	return &AzureMachine{
		Spec: AzureMachineSpec{
			SSHPublicKey: sshPublicKey,
			OSDisk:       generateValidOSDisk(),
			Image: &Image{
				SharedGallery: &AzureSharedGalleryImage{
					SubscriptionID: "SUB123",
					ResourceGroup:  "RG123",
					Name:           "NAME123",
					Gallery:        "GALLERY1",
					Version:        "1.0.0",
				},
			},
		},
	}
}
