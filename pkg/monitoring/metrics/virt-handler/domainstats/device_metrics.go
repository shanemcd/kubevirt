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
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var (
	guestDeviceDriverDateSeconds = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_guest_device_driver_date_seconds",
			Help: "Guest device driver date as Unix epoch seconds, reported by the QEMU guest agent guest-get-devices command.",
		},
	)
)

type deviceMetrics struct{}

func (deviceMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		guestDeviceDriverDateSeconds,
	}
}

func (deviceMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	for _, device := range vmiReport.vmiStats.DeviceStats {
		labels := map[string]string{
			"driver_name":    device.DriverName,
			"driver_version": device.DriverVersion,
			"device_id":      fmt.Sprintf("%x", device.ID.DeviceID),
		}

		crs = append(crs,
			vmiReport.newCollectorResultWithLabels(
				guestDeviceDriverDateSeconds,
				float64(device.DriverDate/1e9),
				labels,
			),
		)
	}

	return crs
}
