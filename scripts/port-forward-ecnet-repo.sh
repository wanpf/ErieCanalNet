#!/bin/bash

# shellcheck disable=SC1091
set -aueo pipefail

K8S_NAMESPACE=ecnet-system

ECNET_POD=$(kubectl get pods -n "$K8S_NAMESPACE" --no-headers  --selector app=ecnet-controller | awk 'NR==1{print $1}')

kubectl port-forward -n "$K8S_NAMESPACE" "$ECNET_POD" 6060:6060 --address 0.0.0.0
