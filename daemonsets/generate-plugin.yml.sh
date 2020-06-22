#!/usr/bin/env bash

: ${NVIDIA_DEVICE_PLUGIN_YAML_SCRIPT:="https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/daemonsets/nvidia-device-plugin.yml.go"}

ARGV=${@} # This needs to be kept outside of the call below to avoid problems with expanding it inside a string

docker run golang bash -c " \
    curl -s ${NVIDIA_DEVICE_PLUGIN_YAML_SCRIPT} -o plugin.yml.go; \
    go run plugin.yml.go ${ARGV} \
"
