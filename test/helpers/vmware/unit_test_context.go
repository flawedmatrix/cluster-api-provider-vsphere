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

package vmware

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	vmoprv1 "github.com/vmware-tanzu/vm-operator-api/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vmwarev1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/vmware/v1beta1"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/fake"
	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/vmware"
)

// UnitTestContextForController is used for unit testing controllers.
type UnitTestContextForController struct {
	// GuestClusterContext is initialized with fake.NewGuestClusterContext
	// and is used for unit testing.
	*vmware.GuestClusterContext

	// Key may be used to lookup Ctx.Cluster with Ctx.Client.Get.
	Key client.ObjectKey

	VirtualMachineImage *vmoprv1.VirtualMachineImage
}

// NewUnitTestContextForController returns a new UnitTestContextForController
// with an optional prototype cluster for unit testing controllers that do not
// invoke the VSphereCluster spec controller.
func NewUnitTestContextForController( /*newReconcilerFn NewReconcilerFunc, */ namespace string, vSphereCluster *vmwarev1.VSphereCluster,
	prototypeCluster bool, initObjects, gcInitObjects []client.Object) *UnitTestContextForController {
	ctx := &UnitTestContextForController{
		GuestClusterContext: fake.NewGuestClusterContext(fake.NewVmwareClusterContext(
			fake.NewControllerContext(
				fake.NewControllerManagerContext(initObjects...)), namespace, vSphereCluster), prototypeCluster, gcInitObjects...),
	}
	ctx.Key = client.ObjectKey{Namespace: ctx.VSphereCluster.Namespace, Name: ctx.VSphereCluster.Name}

	CreatePrototypePrereqs(ctx, ctx.ControllerManagerContext)

	return ctx
}

func CreatePrototypePrereqs(_ *UnitTestContextForController, ctx *context.ControllerManagerContext) {
	By("Creating a prototype VirtualMachineClass", func() {
		virtualMachineClass := FakeVirtualMachineClass()
		virtualMachineClass.Name = "small"
		Expect(ctx.Client.Create(ctx, virtualMachineClass)).To(Succeed())
		virtualMachineClassKey := client.ObjectKey{Name: virtualMachineClass.Name}
		Eventually(func() error {
			return ctx.Client.Get(ctx, virtualMachineClassKey, virtualMachineClass)
		}).Should(Succeed())
	})
}

func FakeVirtualMachineClass() *vmoprv1.VirtualMachineClass {
	return &vmoprv1.VirtualMachineClass{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
		Spec: vmoprv1.VirtualMachineClassSpec{
			Hardware: vmoprv1.VirtualMachineClassHardware{
				Cpus:   int64(2),
				Memory: resource.MustParse("4Gi"),
			},
			Policies: vmoprv1.VirtualMachineClassPolicies{
				Resources: vmoprv1.VirtualMachineClassResources{
					Requests: vmoprv1.VirtualMachineResourceSpec{
						Cpu:    resource.MustParse("2Gi"),
						Memory: resource.MustParse("4Gi"),
					},
					Limits: vmoprv1.VirtualMachineResourceSpec{
						Cpu:    resource.MustParse("2Gi"),
						Memory: resource.MustParse("4Gi"),
					},
				},
			},
		},
	}
}
