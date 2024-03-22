#!/usr/bin/env bash

helm diff upgrade --namespace kube-system \
karpenter oci://$ECR_ACCOUNT_ID.dkr.ecr.$ECR_REGION.amazonaws.com/karpenter/snapshot/karpenter \
--version 0-$(git rev-parse HEAD) \
--reuse-values --three-way-merge --detailed-exitcode
