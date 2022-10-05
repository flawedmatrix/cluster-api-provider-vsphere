/*
Copyright 2022 The Kubernetes Authors.

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

package govmomi

import (
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1alpha1"
	ipamv1a1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1alpha1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
)

func Test_reconcileIPAddressClaims_ShouldGenerateIPAddressClaims(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = ipamv1a1.AddToScheme(scheme)

	ctx := emptyVirtualMachineContext()
	ctx.Client = fake.NewClientBuilder().WithScheme(scheme).Build()

	myApiGroup := "my-pool-api-group"

	t.Run("when a device has a IPAddressPool", func(t *testing.T) {
		ctx.VSphereVM = &infrav1.VSphereVM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vsphereVM1",
				Namespace: "my-namespace",
			},
			Spec: infrav1.VSphereVMSpec{
				VirtualMachineCloneSpec: infrav1.VirtualMachineCloneSpec{
					Network: infrav1.NetworkSpec{
						Devices: []infrav1.NetworkDeviceSpec{
							{
								FromPools: []corev1.TypedLocalObjectReference{
									{
										APIGroup: &myApiGroup,
										Name:     "my-pool-1",
										Kind:     "my-pool-kind",
									},
								},
							},
							{
								FromPools: []corev1.TypedLocalObjectReference{
									{
										APIGroup: &myApiGroup,
										Name:     "my-pool-2",
										Kind:     "my-pool-kind",
									},
									{
										APIGroup: &myApiGroup,
										Name:     "my-pool-3",
										Kind:     "my-pool-kind",
									},
								},
							},
						},
					},
				},
			},
		}

		vms := &VMService{}
		g := NewWithT(t)

		reconciled, err := vms.reconcileIPAddressClaims(ctx)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(reconciled).To(BeTrue())

		ipAddrClaimKey := apitypes.NamespacedName{
			Name:      "vsphereVM1-0-0",
			Namespace: "my-namespace",
		}
		ipAddrClaim := &ipamv1a1.IPAddressClaim{}
		ctx.Client.Get(ctx, ipAddrClaimKey, ipAddrClaim)
		g.Expect(ipAddrClaim.Spec.PoolRef.Name).To(Equal("my-pool-1"))

		ipAddrClaimKey = apitypes.NamespacedName{
			Name:      "vsphereVM1-1-0",
			Namespace: "my-namespace",
		}
		ipAddrClaim = &ipamv1a1.IPAddressClaim{}
		ctx.Client.Get(ctx, ipAddrClaimKey, ipAddrClaim)
		g.Expect(ipAddrClaim.Spec.PoolRef.Name).To(Equal("my-pool-2"))

		ipAddrClaimKey = apitypes.NamespacedName{
			Name:      "vsphereVM1-1-1",
			Namespace: "my-namespace",
		}
		ipAddrClaim = &ipamv1a1.IPAddressClaim{}
		ctx.Client.Get(ctx, ipAddrClaimKey, ipAddrClaim)
		g.Expect(ipAddrClaim.Spec.PoolRef.Name).To(Equal("my-pool-3"))

		// Ensure that duplicate claims are not created
		reconciled, err = vms.reconcileIPAddressClaims(ctx)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(reconciled).To(BeTrue())

		ipAddrClaims := &ipamv1a1.IPAddressClaimList{}
		ctx.Client.List(ctx, ipAddrClaims)
		g.Expect(ipAddrClaims.Items).To(HaveLen(3))

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// for the WaitingForIPAddress reason.
		claimedCondition := conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.WaitingForIPAddressReason))
	})
}

func Test_reconcileIPAddresses_ShouldUpdateVMDevicesWithAddresses(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = ipamv1a1.AddToScheme(scheme)

	ctx := emptyVirtualMachineContext()
	ctx.Client = fake.NewClientBuilder().WithScheme(scheme).Build()

	myApiGroup := "my-pool-api-group"

	claim1 := &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-0",
			Namespace: "my-namespace",
		},
	}

	claim2 := &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-1",
			Namespace: "my-namespace",
		},
	}

	claim3 := &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-2",
			Namespace: "my-namespace",
		},
	}

	address1 := &ipamv1.IPAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-0-address0",
			Namespace: "my-namespace",
		},
		Spec: ipamv1.IPAddressSpec{
			Address: "10.0.0.50",
			Prefix:  24,
			Gateway: "10.0.0.1",
		},
	}
	address2 := &ipamv1.IPAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-1-address1",
			Namespace: "my-namespace",
		},
		Spec: ipamv1.IPAddressSpec{
			Address: "10.0.1.50",
			Prefix:  30,
			Gateway: "10.0.0.1",
		},
	}

	address3 := &ipamv1.IPAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-2-address2",
			Namespace: "my-namespace",
		},
		Spec: ipamv1.IPAddressSpec{
			Address: "fe80::cccc:12",
			Prefix:  64,
			Gateway: "fe80::cccc:1",
		},
	}

	t.Run("when a device has a IPAddressPool", func(t *testing.T) {
		ctx.VSphereVM = &infrav1.VSphereVM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vsphereVM1",
				Namespace: "my-namespace",
			},
			Spec: infrav1.VSphereVMSpec{
				VirtualMachineCloneSpec: infrav1.VirtualMachineCloneSpec{
					Network: infrav1.NetworkSpec{
						Devices: []infrav1.NetworkDeviceSpec{
							{
								FromPools: []corev1.TypedLocalObjectReference{
									{
										APIGroup: &myApiGroup,
										Name:     "my-pool-1",
										Kind:     "my-pool-kind",
									},
									{
										APIGroup: &myApiGroup,
										Name:     "my-pool-1",
										Kind:     "my-pool-kind",
									},
									{
										APIGroup: &myApiGroup,
										Name:     "my-pool-ipv6",
										Kind:     "my-pool-kind",
									},
								},
							},
						},
					},
				},
			},
		}

		vms := &VMService{}
		g := NewWithT(t)

		// Creates ip address claims
		ctx.Client.Create(ctx, claim1)
		ctx.Client.Create(ctx, claim2)
		ctx.Client.Create(ctx, claim3)

		// IP provider has not provided Addresses yet
		reconciled, err := vms.reconcileIPAddresses(ctx)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(reconciled).To(BeFalse())

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// for the WaitingForIPAddress reason.
		claimedCondition := conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.WaitingForIPAddressReason))
		g.Expect(claimedCondition.Message).To(Equal("Waiting for IPAddressClaim to have an IPAddress bound"))
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionFalse))

		// Simulate IP provider reconciling claim
		ctx.Client.Create(ctx, address1)
		ctx.Client.Create(ctx, address2)
		ctx.Client.Create(ctx, address3)

		ipAddrClaim := &ipamv1.IPAddressClaim{}
		ipAddrClaimKey := apitypes.NamespacedName{
			Namespace: ctx.VSphereVM.Namespace,
			Name:      "vsphereVM1-0-0",
		}
		err = ctx.Client.Get(ctx, ipAddrClaimKey, ipAddrClaim)
		g.Expect(err).NotTo(HaveOccurred())

		ipAddrClaim.Status.AddressRef.Name = "vsphereVM1-0-0-address0"

		ctx.Client.Update(ctx, ipAddrClaim)

		ipAddrClaimKey = apitypes.NamespacedName{
			Namespace: ctx.VSphereVM.Namespace,
			Name:      "vsphereVM1-0-1",
		}
		err = ctx.Client.Get(ctx, ipAddrClaimKey, ipAddrClaim)
		g.Expect(err).NotTo(HaveOccurred())

		ipAddrClaim.Status.AddressRef.Name = "vsphereVM1-0-1-address1"

		ctx.Client.Update(ctx, ipAddrClaim)

		ipAddrClaimKey = apitypes.NamespacedName{
			Namespace: ctx.VSphereVM.Namespace,
			Name:      "vsphereVM1-0-2",
		}
		err = ctx.Client.Get(ctx, ipAddrClaimKey, ipAddrClaim)
		g.Expect(err).NotTo(HaveOccurred())

		ipAddrClaim.Status.AddressRef.Name = "vsphereVM1-0-2-address2"

		ctx.Client.Update(ctx, ipAddrClaim)

		// Now that claims are fulfilled, reconciling should update
		// ipAddrs on network spec
		reconciled, err = vms.reconcileIPAddresses(ctx)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(reconciled).To(BeTrue())
		g.Expect(ctx.VSphereVM.Spec.Network.Devices).To(HaveLen(1))
		g.Expect(ctx.VSphereVM.Spec.Network.Devices[0].IPAddrs).To(HaveLen(3))
		g.Expect(ctx.VSphereVM.Spec.Network.Devices[0].IPAddrs[0]).To(Equal("10.0.0.50/24"))
		g.Expect(ctx.VSphereVM.Spec.Network.Devices[0].Gateway4).To(Equal("10.0.0.1"))
		g.Expect(ctx.VSphereVM.Spec.Network.Devices[0].IPAddrs[1]).To(Equal("10.0.1.50/30"))
		g.Expect(ctx.VSphereVM.Spec.Network.Devices[0].Gateway4).To(Equal("10.0.0.1"))
		g.Expect(ctx.VSphereVM.Spec.Network.Devices[0].IPAddrs[2]).To(Equal("fe80::cccc:12/64"))
		g.Expect(ctx.VSphereVM.Spec.Network.Devices[0].Gateway6).To(Equal("fe80::cccc:1"))
		claimedCondition = conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionTrue))
	})
}

func Test_reconcileIPAddresses_ShouldUpdateTheStatusOnValidationIssues(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = ipamv1a1.AddToScheme(scheme)

	ctx := emptyVirtualMachineContext()
	ctx.Client = fake.NewClientBuilder().WithScheme(scheme).Build()

	myApiGroup := "my-pool-api-group"

	claim1 := &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-0",
			Namespace: "my-namespace",
		},
		Status: ipamv1.IPAddressClaimStatus{
			AddressRef: corev1.LocalObjectReference{
				Name: "vsphereVM1-0-0",
			},
		},
	}

	claim2 := &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-1",
			Namespace: "my-namespace",
		},
		Status: ipamv1.IPAddressClaimStatus{
			AddressRef: corev1.LocalObjectReference{
				Name: "vsphereVM1-0-1",
			},
		},
	}

	address1 := &ipamv1.IPAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-0",
			Namespace: "my-namespace",
		},
		Spec: ipamv1.IPAddressSpec{
			Address: "10.0.1.50",
			Prefix:  24,
			Gateway: "10.0.0.1",
		},
	}

	address2 := &ipamv1.IPAddress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vsphereVM1-0-1",
			Namespace: "my-namespace",
		},
		Spec: ipamv1.IPAddressSpec{
			Address: "10.0.1.51",
			Prefix:  24,
			Gateway: "10.0.0.1",
		},
	}

	t.Run("when a device has a IPAddressPool", func(t *testing.T) {
		ctx.VSphereVM = &infrav1.VSphereVM{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vsphereVM1",
				Namespace: "my-namespace",
			},
			Spec: infrav1.VSphereVMSpec{
				VirtualMachineCloneSpec: infrav1.VirtualMachineCloneSpec{
					Network: infrav1.NetworkSpec{
						Devices: []infrav1.NetworkDeviceSpec{
							{
								FromPools: []corev1.TypedLocalObjectReference{
									{
										APIGroup: &myApiGroup,
										Name:     "my-pool-1",
										Kind:     "my-pool-kind",
									},
									{
										APIGroup: &myApiGroup,
										Name:     "my-pool-2",
										Kind:     "my-pool-kind",
									},
								},
							},
						},
					},
				},
			},
		}

		vms := &VMService{}
		g := NewWithT(t)

		// Creates ip address claims
		ctx.Client.Create(ctx, claim1)
		ctx.Client.Create(ctx, claim2)

		// Simulate an invalid ip address was provided: the address is empty
		address1.Spec.Address = ""
		ctx.Client.Create(ctx, address1)
		ctx.Client.Create(ctx, address2)

		// IP provider has not provided Addresses yet
		reconciled, err := vms.reconcileIPAddresses(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(reconciled).To(BeFalse())

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// because the simulated ip address is missing the spec address.
		claimedCondition := conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.IPAddressInvalidReason))
		g.Expect(claimedCondition.Message).To(Equal("IPAddress my-namespace/vsphereVM1-0-0 has invalid ip address: \"/24\""))
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionFalse))

		// Simulate an invalid ip address was provided: the address is not a valid ip
		address1.Spec.Address = "invalid-ip"
		ctx.Client.Update(ctx, address1)

		// IP provider has not provided Addresses yet
		reconciled, err = vms.reconcileIPAddresses(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(reconciled).To(BeFalse())

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// because the simulated ip address is missing the spec address.
		claimedCondition = conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.IPAddressInvalidReason))
		g.Expect(claimedCondition.Message).To(Equal("IPAddress my-namespace/vsphereVM1-0-0 has invalid ip address: \"invalid-ip/24\""))
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionFalse))

		// Simulate an invalid prefix address was provided: the prefix is out of bounds
		address1.Spec.Address = "10.0.1.50"
		address1.Spec.Prefix = 200
		ctx.Client.Update(ctx, address1)

		// IP provider has not provided Addresses yet
		reconciled, err = vms.reconcileIPAddresses(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(reconciled).To(BeFalse())

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// because the simulated ip address is missing the spec address.
		claimedCondition = conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.IPAddressInvalidReason))
		g.Expect(claimedCondition.Message).To(Equal("IPAddress my-namespace/vsphereVM1-0-0 has invalid ip address: \"10.0.1.50/200\""))
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionFalse))

		// Simulate an invalid gateway was provided: the gateway is an invalid ip
		address1.Spec.Prefix = 24
		address1.Spec.Gateway = "invalid-gateway"
		ctx.Client.Update(ctx, address1)

		// IP provider has not provided Addresses yet
		reconciled, err = vms.reconcileIPAddresses(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(reconciled).To(BeFalse())

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// because the simulated ip address is missing the spec address.
		claimedCondition = conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.IPAddressInvalidReason))
		g.Expect(claimedCondition.Message).To(Equal("IPAddress my-namespace/vsphereVM1-0-0 has invalid gateway: \"invalid-gateway\""))
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionFalse))

		// Simulate mismatch address and gateways were provided
		address1.Spec.Address = "10.0.1.50"
		address1.Spec.Gateway = "fd00::1"
		ctx.Client.Update(ctx, address1)

		// IP provider has not provided Addresses yet
		reconciled, err = vms.reconcileIPAddresses(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(reconciled).To(BeFalse())

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// because the simulated ip address is missing the spec address.
		claimedCondition = conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.IPAddressInvalidReason))
		g.Expect(claimedCondition.Message).To(Equal("IPAddress my-namespace/vsphereVM1-0-0 has mismatched gateway and address IP families"))
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionFalse))

		// Simulate mismatch address and gateways were provided
		address1.Spec.Address = "fd00:cccc::1"
		address1.Spec.Gateway = "10.0.0.1"
		ctx.Client.Update(ctx, address1)

		// IP provider has not provided Addresses yet
		reconciled, err = vms.reconcileIPAddresses(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(reconciled).To(BeFalse())

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// because the simulated ip address is missing the spec address.
		claimedCondition = conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.IPAddressInvalidReason))
		g.Expect(claimedCondition.Message).To(Equal("IPAddress my-namespace/vsphereVM1-0-0 has mismatched gateway and address IP families"))
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionFalse))

		// Simulate mulitple gateways were provided
		address1.Spec.Address = "10.0.1.50"
		address1.Spec.Gateway = "10.0.0.1"
		ctx.Client.Update(ctx, address1)
		address2.Spec.Address = "10.0.1.51"
		address2.Spec.Gateway = "10.0.0.2"
		ctx.Client.Update(ctx, address2)

		// IP provider has not provided Addresses yet
		reconciled, err = vms.reconcileIPAddresses(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(reconciled).To(BeFalse())

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// because the simulated ip address is missing the spec address.
		claimedCondition = conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.IPAddressInvalidReason))
		g.Expect(claimedCondition.Message).To(Equal("The IPv4 IPAddresses assigned to the same device (index 0) do not have the same gateway"))
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionFalse))

		// Simulate mulitple gateways were provided
		address1.Spec.Address = "fd00:cccc::2"
		address1.Spec.Gateway = "fd00::1"
		ctx.Client.Update(ctx, address1)
		address2.Spec.Address = "fd00:cccc::3"
		address2.Spec.Gateway = "fd00::2"
		ctx.Client.Update(ctx, address2)

		// IP provider has not provided Addresses yet
		reconciled, err = vms.reconcileIPAddresses(ctx)
		g.Expect(err).To(HaveOccurred())
		g.Expect(reconciled).To(BeFalse())

		// Ensure that the VM has a IPAddressClaimed condition set to False
		// because the simulated ip address is missing the spec address.
		claimedCondition = conditions.Get(ctx.VSphereVM, infrav1.IPAddressClaimedCondition)
		g.Expect(claimedCondition).NotTo(BeNil())
		g.Expect(claimedCondition.Reason).To(Equal(infrav1.IPAddressInvalidReason))
		g.Expect(claimedCondition.Message).To(Equal("The IPv6 IPAddresses assigned to the same device (index 0) do not have the same gateway"))
		g.Expect(claimedCondition.Status).To(Equal(corev1.ConditionFalse))
	})
}

func emptyVirtualMachineContext() *virtualMachineContext {
	return &virtualMachineContext{
		VMContext: context.VMContext{
			Logger: logr.Discard(),
			ControllerContext: &context.ControllerContext{
				ControllerManagerContext: &context.ControllerManagerContext{},
			},
		},
	}
}
