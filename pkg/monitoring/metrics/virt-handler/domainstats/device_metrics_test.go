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

package domainstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("device metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		vmiStats := &VirtualMachineInstanceStats{
			DeviceStats: []api.Device{
				{
					DriverName:    "Red Hat VirtIO Ethernet Adapter",
					DriverVersion: "100.95.104.26200",
					DriverDate:    1721001600000000000,
					ID: api.DeviceID{
						DeviceID: 4161,
						VendorID: 6900,
						Type:     "pci",
					},
				},
			},
		}

		vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)

		It("should collect kubevirt_vmi_guest_device_driver_date_seconds", func() {
			crs := deviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(guestDeviceDriverDateSeconds, 1721001600.0)))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue("device_id", "1041"))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue("driver_name", "Red Hat VirtIO Ethernet Adapter"))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue("driver_version", "100.95.104.26200"))
		})

		It("result should be empty if no devices are present", func() {
			emptyReport := newVirtualMachineInstanceReport(vmi, &VirtualMachineInstanceStats{})
			crs := deviceMetrics{}.Collect(emptyReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
