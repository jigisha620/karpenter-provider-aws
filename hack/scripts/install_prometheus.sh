# Remove service account annotation when dropping support for 1.23
helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
-n prometheus \
-f ./.github/actions/e2e/install-prometheus/values.yaml \
--set prometheus.prometheusSpec.remoteWrite[0].url=https://aps-workspaces.$PROMETHEUS_REGION.amazonaws.com/workspaces/$WORKSPACE_ID/api/v1/remote_write \
--set prometheus.prometheusSpec.remoteWrite[0].sigv4.region=$PROMETHEUS_REGION \
--set prometheus.serviceAccount.annotations."eks\.amazonaws\.com/role-arn"="arn:aws:iam::$ACCOUNT_ID:role/prometheus-irsa-$CLUSTER_NAME" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[0].targetLabel=metrics_path" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[0].action=replace" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[0].sourceLabels[0]=__metrics_path__" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[1].targetLabel=clusterName" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[1].replacement=$CLUSTER_NAME" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[2].targetLabel=gitRef" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[2].replacement=$(git rev-parse HEAD)" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[3].targetLabel=mostRecentTag" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[3].replacement=$(git describe --abbrev=0 --tags)" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[4].targetLabel=commitsAfterTag" \
--set "kubelet.serviceMonitor.cAdvisorRelabelings[4].replacement=\"$(git describe --tags | cut -d '-' -f 2)\"" \
--wait