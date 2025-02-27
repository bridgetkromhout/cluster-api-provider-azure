/*
Copyright 2020 The Kubernetes Authors.

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

package converters

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-11-01/compute"
	"k8s.io/utils/pointer"
	azprovider "sigs.k8s.io/cloud-provider-azure/pkg/provider"
	infrav1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-azure/azure"
)

// SDKToVMSS converts an Azure SDK VirtualMachineScaleSet to the AzureMachinePool type.
func SDKToVMSS(sdkvmss compute.VirtualMachineScaleSet, sdkinstances []compute.VirtualMachineScaleSetVM) *azure.VMSS {
	vmss := &azure.VMSS{
		ID:    pointer.StringDeref(sdkvmss.ID, ""),
		Name:  pointer.StringDeref(sdkvmss.Name, ""),
		State: infrav1.ProvisioningState(pointer.StringDeref(sdkvmss.ProvisioningState, "")),
	}

	if sdkvmss.Sku != nil {
		vmss.Sku = pointer.StringDeref(sdkvmss.Sku.Name, "")
		vmss.Capacity = pointer.Int64Deref(sdkvmss.Sku.Capacity, 0)
	}

	if sdkvmss.Zones != nil && len(*sdkvmss.Zones) > 0 {
		vmss.Zones = azure.StringSlice(sdkvmss.Zones)
	}

	if len(sdkvmss.Tags) > 0 {
		vmss.Tags = MapToTags(sdkvmss.Tags)
	}

	if len(sdkinstances) > 0 {
		vmss.Instances = make([]azure.VMSSVM, len(sdkinstances))
		for i, vm := range sdkinstances {
			vmss.Instances[i] = *SDKToVMSSVM(vm)
		}
	}

	if sdkvmss.VirtualMachineProfile != nil &&
		sdkvmss.VirtualMachineProfile.StorageProfile != nil &&
		sdkvmss.VirtualMachineProfile.StorageProfile.ImageReference != nil {
		imageRef := sdkvmss.VirtualMachineProfile.StorageProfile.ImageReference
		vmss.Image = SDKImageToImage(imageRef, sdkvmss.Plan != nil)
	}

	return vmss
}

// SDKVMToVMSSVM converts an Azure SDK VM to a VMSS VM.
func SDKVMToVMSSVM(sdkInstance compute.VirtualMachine) *azure.VMSSVM {
	instance := azure.VMSSVM{
		ID: pointer.StringDeref(sdkInstance.ID, ""),
	}

	if sdkInstance.VirtualMachineProperties == nil {
		return &instance
	}

	instance.State = infrav1.Creating
	if sdkInstance.ProvisioningState != nil {
		instance.State = infrav1.ProvisioningState(pointer.StringDeref(sdkInstance.ProvisioningState, ""))
	}

	if sdkInstance.OsProfile != nil && sdkInstance.OsProfile.ComputerName != nil {
		instance.Name = *sdkInstance.OsProfile.ComputerName
	}

	if sdkInstance.StorageProfile != nil && sdkInstance.StorageProfile.ImageReference != nil {
		imageRef := sdkInstance.StorageProfile.ImageReference
		instance.Image = SDKImageToImage(imageRef, sdkInstance.Plan != nil)
	}

	if sdkInstance.Zones != nil && len(*sdkInstance.Zones) > 0 {
		// An instance should have only 1 zone, so use the first item of the slice.
		instance.AvailabilityZone = azure.StringSlice(sdkInstance.Zones)[0]
	}

	return &instance
}

// SDKToVMSSVM converts an Azure SDK VirtualMachineScaleSetVM into an infrav1exp.VMSSVM.
func SDKToVMSSVM(sdkInstance compute.VirtualMachineScaleSetVM) *azure.VMSSVM {
	// Convert resourceGroup Name ID ( ProviderID in capz objects )
	var convertedID string
	convertedID, err := azprovider.ConvertResourceGroupNameToLower(pointer.StringDeref(sdkInstance.ID, ""))
	if err != nil {
		convertedID = pointer.StringDeref(sdkInstance.ID, "")
	}

	instance := azure.VMSSVM{
		ID:         convertedID,
		InstanceID: pointer.StringDeref(sdkInstance.InstanceID, ""),
	}

	if sdkInstance.VirtualMachineScaleSetVMProperties == nil {
		return &instance
	}

	instance.State = infrav1.Creating
	if sdkInstance.ProvisioningState != nil {
		instance.State = infrav1.ProvisioningState(pointer.StringDeref(sdkInstance.ProvisioningState, ""))
	}

	if sdkInstance.OsProfile != nil && sdkInstance.OsProfile.ComputerName != nil {
		instance.Name = *sdkInstance.OsProfile.ComputerName
	}

	if sdkInstance.Resources != nil {
		for _, r := range *sdkInstance.Resources {
			if r.ProvisioningState != nil && r.Name != nil &&
				(*r.Name == azure.BootstrappingExtensionLinux || *r.Name == azure.BootstrappingExtensionWindows) {
				instance.BootstrappingState = infrav1.ProvisioningState(pointer.StringDeref(r.ProvisioningState, ""))
				break
			}
		}
	}

	if sdkInstance.StorageProfile != nil && sdkInstance.StorageProfile.ImageReference != nil {
		imageRef := sdkInstance.StorageProfile.ImageReference
		instance.Image = SDKImageToImage(imageRef, sdkInstance.Plan != nil)
	}

	if sdkInstance.Zones != nil && len(*sdkInstance.Zones) > 0 {
		// an instance should only have 1 zone, so we select the first item of the slice
		instance.AvailabilityZone = azure.StringSlice(sdkInstance.Zones)[0]
	}

	return &instance
}

// SDKImageToImage converts a SDK image reference to infrav1.Image.
func SDKImageToImage(sdkImageRef *compute.ImageReference, isThirdPartyImage bool) infrav1.Image {
	return infrav1.Image{
		ID: sdkImageRef.ID,
		Marketplace: &infrav1.AzureMarketplaceImage{
			ImagePlan: infrav1.ImagePlan{
				Publisher: pointer.StringDeref(sdkImageRef.Publisher, ""),
				Offer:     pointer.StringDeref(sdkImageRef.Offer, ""),
				SKU:       pointer.StringDeref(sdkImageRef.Sku, ""),
			},
			Version:         pointer.StringDeref(sdkImageRef.Version, ""),
			ThirdPartyImage: isThirdPartyImage,
		},
	}
}

// GetOrchestrationMode returns the compute.OrchestrationMode for the given infrav1.OrchestrationModeType.
func GetOrchestrationMode(modeType infrav1.OrchestrationModeType) compute.OrchestrationMode {
	if modeType == infrav1.FlexibleOrchestrationMode {
		return compute.OrchestrationModeFlexible
	}
	return compute.OrchestrationModeUniform
}
