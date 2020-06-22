package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/template"
)

var legacyDaemonsetApiFlag = flag.Bool(
	"legacy-daemonset-api",
	false,
	"indicates whether we want to use the legacy daemonset API version 'extensions/v1beta1' or not\n"+
		"(default 'false')")

var imageTagFlag = flag.String(
	"image-tag",
	"latest",
	"pass the desired docker image TAG for the device plugin from docker hub\n"+
		"https://hub.docker.com/r/nvidia/k8s-device-plugin")

var migStrategyFlag = flag.String(
	"mig-strategy",
	"none",
	"pass the desired strategy for exposing MIG devices on GPUs that support it\n"+
		"[none | single | mixed]")

var withCPUManagerFlag = flag.Bool(
	"compat-with-cpu-manager",
	false,
	"indicates whether we run with escalated privileges to be compatible with the CPUManager or not\n"+
		"(default 'false')")

const PluginTemplate = `
# Copyright (c) 2020, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

apiVersion: {{ .ApiVersion }}
kind: DaemonSet
metadata:
  name: nvidia-device-plugin-daemonset
  namespace: kube-system
spec:
  {{ .Selector }}
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      # This annotation is deprecated. Kept here for backward compatibility
      # See https://kubernetes.io/docs/tasks/administer-cluster/guaranteed-scheduling-critical-addon-pods/
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ""
      labels:
        name: nvidia-device-plugin-ds
    spec:
      tolerations:
      # This toleration is deprecated. Kept here for backward compatibility
      # See https://kubernetes.io/docs/tasks/administer-cluster/guaranteed-scheduling-critical-addon-pods/
      - key: CriticalAddonsOnly
        operator: Exists
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
      # Mark this pod as a critical add-on; when enabled, the critical add-on
      # scheduler reserves resources for critical add-on pods so that they can
      # be rescheduled after a failure.
      # See https://kubernetes.io/docs/tasks/administer-cluster/guaranteed-scheduling-critical-addon-pods/
      priorityClassName: "system-node-critical"
      containers:
      - image: nvidia/k8s-device-plugin:{{ .ImageTag }}
        name: nvidia-device-plugin-ctr
        args: [{{ range $arg := .Args }} "{{ $arg }}", {{ end }}]
        {{ .SecurityContext }}
        volumeMounts:
          - name: device-plugin
            mountPath: /var/lib/kubelet/device-plugins
      volumes:
        - name: device-plugin
          hostPath:
            path: /var/lib/kubelet/device-plugins
`

const defaultApiVersion = "apps/v1"
const defaultSelector = `
  selector:
    matchLabels:
      name: nvidia-device-plugin-ds
`
const legacyApiVersion = "extensions/v1beta1"
const legacySelector = ""

const SecurityContextWithCPUManager = `
        securityContext:
          privileged: true
`

const SecurityContextWithoutCPUManager = `
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop: ["ALL"]
`

type Customizations struct {
	ApiVersion      string
	Selector        string
	ImageTag        string
	Args            []string
	SecurityContext string
}

func main() {
	flag.Parse()

	customizations := Customizations{
		ApiVersion:      defaultApiVersion,
		Selector:        defaultSelector,
		ImageTag:        *imageTagFlag,
		Args:            []string{fmt.Sprintf("--mig-strategy=%s", *migStrategyFlag)},
		SecurityContext: SecurityContextWithoutCPUManager,
	}

	if *legacyDaemonsetApiFlag {
		customizations.ApiVersion = legacyApiVersion
		customizations.Selector = legacySelector
	}

	if *withCPUManagerFlag {
		customizations.Args = append(customizations.Args, "--pass-device-specs")
		customizations.SecurityContext = SecurityContextWithCPUManager
	}

	t := template.Must(template.New("plugin").Parse(PluginTemplate))
	err := t.Execute(os.Stdout, customizations)
	if err != nil {
		log.Fatalf("%v", err)
	}
}
