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

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	ipamv1a1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
)

func Test_ShouldGenerateIPAddressClaims(t *testing.T) {
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
		g.Expect(reconciled).To(BeFalse())

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
		g.Expect(reconciled).To(BeFalse())

		ipAddrClaims := &ipamv1a1.IPAddressClaimList{}
		ctx.Client.List(ctx, ipAddrClaims)
		g.Expect(ipAddrClaims.Items).To(HaveLen(3))

		// simulate that the pool associated a ipaddr to the claim

		// reconcile again
		// expect it returns true
		// see that the vm device spec has ipAddr has the iP ? // not part of current commit/effort?
		//

	})
}

func emptyVirtualMachineContext() *virtualMachineContext {
	return &virtualMachineContext{
		VMContext: context.VMContext{
			ControllerContext: &context.ControllerContext{
				ControllerManagerContext: &context.ControllerManagerContext{},
			},
		},
	}
}
