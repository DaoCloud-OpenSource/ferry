#!/usr/bin/env bash
# Copyright 2022 FerryProxy Authors.
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


CURRENT="$(dirname "${BASH_SOURCE}")"
ROOT="$(realpath "${CURRENT}/../..")"
ENVIRONMENT_NAME="${CURRENT##*/}"

ENVIRONMENT_DIR="${ROOT}/environments/${ENVIRONMENT_NAME}"
KUBECONFIG_DIR="${ROOT}/kubeconfigs"

HOST_IP="$(${ROOT}/hack/host-docker-internal.sh)"
echo "Host IP: ${HOST_IP}"

export KUBECONFIG

echo "::group::Control plane initialization"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl control-plane init --control-plane-reachable=false
ferryctl control-plane init --control-plane-reachable=false
echo "::endgroup::"

echo "::group::Data plane cluster-1 join"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl control-plane join cluster-1 "--data-plane-tunnel-address=${HOST_IP}:31001" --control-plane-reachable=false
SEND_TO_CLUSTER_1="$(ferryctl control-plane join cluster-1 "--data-plane-tunnel-address=${HOST_IP}:31001" --control-plane-reachable=false 2>/dev/null)"
echo "::endgroup::"

echo "::group::Data plane cluster-1 join"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_1}"
SEND_TO_CONTROL_PLANE="$(eval "${SEND_TO_CLUSTER_1}" 2>/dev/null)"
echo "::endgroup::"

echo "::group::Controll plane confirm cluster-1 join"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CONTROL_PLANE}"
eval "${SEND_TO_CONTROL_PLANE}"
echo "::endgroup::"
