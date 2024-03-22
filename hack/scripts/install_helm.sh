#!/usr/bin/env bash

TEMPDIR=$(mktemp -d)
curl -fsSL -o "${TEMPDIR}/get_helm.sh" https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 "${TEMPDIR}/get_helm.sh"
"${TEMPDIR}/get_helm.sh" --version "$VERSION"
