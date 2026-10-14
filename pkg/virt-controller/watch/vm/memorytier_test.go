/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package vm

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	resourcev1 "k8s.io/api/resource/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	virtv1 "kubevirt.io/api/core/v1"

	"go.uber.org/mock/gomock"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("Memory tier claim translation", func() {
	newVMWithTier := func(tier string, guest string) *virtv1.VirtualMachine {
		vmi := libvmi.New(libvmi.WithMemoryRequest(guest))
		vmi.Spec.Domain.Memory = &virtv1.Memory{
			Tier:  tier,
			Guest: quantityPtr(guest),
		}
		return &virtv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{Name: "tier-vm", Namespace: metav1.NamespaceDefault, UID: "vm-uid"},
			Spec: virtv1.VirtualMachineSpec{
				Running: boolPtr(false),
				Template: &virtv1.VirtualMachineInstanceTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Name: "tier-vm"},
					Spec:       vmi.Spec,
				},
			},
		}
	}

	DescribeTable("translates the tier into a claim template + VMI reference",
		func(tier, class, guest string) {
			vm := newVMWithTier(tier, guest)
			vmi := SetupVMIFromVM(vm)

			client := fake.NewSimpleClientset()
			controller := &Controller{clientset: newTierTestClient(client)}

			Expect(controller.ensureMemoryTierClaim(vm, vmi)).To(Succeed())

			tmpl, err := client.ResourceV1().ResourceClaimTemplates(metav1.NamespaceDefault).
				Get(context.Background(), "tier-vm-memory-tier", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(tmpl.OwnerReferences).To(HaveLen(1))

			req := tmpl.Spec.Spec.Devices.Requests
			Expect(req).To(HaveLen(1))
			Expect(req[0].Exactly.DeviceClassName).To(Equal(class))
			Expect(req[0].Exactly.AllocationMode).To(Equal(resourcev1.DeviceAllocationModeAll))
			Expect(req[0].Exactly.Capacity.Requests["memorytier.openshift.io/memory"]).To(Equal(resource.MustParse(guest)))

			Expect(vmi.Spec.ResourceClaims).To(HaveLen(1))
			Expect(vmi.Spec.ResourceClaims[0].Name).To(Equal("memory-tier"))
			Expect(*vmi.Spec.ResourceClaims[0].ResourceClaimTemplateName).To(Equal("tier-vm-memory-tier"))
		},
		Entry("DRAM-only", "DRAM-only", "dram-tier", "128Mi"),
		Entry("CXL-only", "CXL-only", "cxl-tier", "256Mi"),
	)

	It("is a no-op without a tier", func() {
		vm := newVMWithTier("", "128Mi")
		vmi := SetupVMIFromVM(vm)
		controller := &Controller{clientset: newTierTestClient(fake.NewSimpleClientset())}

		Expect(controller.ensureMemoryTierClaim(vm, vmi)).To(Succeed())
		Expect(vmi.Spec.ResourceClaims).To(BeEmpty())
	})

	It("rejects an unknown tier", func() {
		vm := newVMWithTier("turbo", "128Mi")
		vmi := SetupVMIFromVM(vm)
		controller := &Controller{clientset: newTierTestClient(fake.NewSimpleClientset())}

		err := controller.ensureMemoryTierClaim(vm, vmi)
		Expect(err).To(MatchError(ContainSubstring("no DeviceClass mapping")))
	})

	It("falls back to the resources request when guest is unset", func() {
		vm := newVMWithTier("DRAM-only", "128Mi")
		vm.Spec.Template.Spec.Domain.Memory.Guest = nil
		vmi := SetupVMIFromVM(vm)
		client := fake.NewSimpleClientset()
		controller := &Controller{clientset: newTierTestClient(client)}

		Expect(controller.ensureMemoryTierClaim(vm, vmi)).To(Succeed())
		tmpl, err := client.ResourceV1().ResourceClaimTemplates(metav1.NamespaceDefault).
			Get(context.Background(), "tier-vm-memory-tier", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		req := tmpl.Spec.Spec.Devices.Requests
		Expect(req[0].Exactly.Capacity.Requests["memorytier.openshift.io/memory"]).To(Equal(resource.MustParse("128Mi")))
	})
})

func quantityPtr(s string) *resource.Quantity {
	q := resource.MustParse(s)
	return &q
}

func boolPtr(b bool) *bool { return &b }

// newTierTestClient wraps a k8s fake clientset in the minimal
// kubecli.KubevirtClient surface the translation needs (the embedded
// kubernetes.Interface).
func newTierTestClient(c *fake.Clientset) kubecli.KubevirtClient {
	virtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))
	virtClient.EXPECT().ResourceV1().Return(c.ResourceV1()).AnyTimes()
	return virtClient
}
