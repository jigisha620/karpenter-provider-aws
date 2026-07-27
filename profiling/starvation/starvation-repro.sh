#!/bin/bash
set -e

PROFILE_DURATION=300  # 5 minutes
PROFILE_OUTPUT="starvation-repro.prof"
PPROF_URL="http://localhost:8080/debug/pprof/profile?seconds=${PROFILE_DURATION}"
REPRO_DIR="${HOME}/Downloads/reproduction"

echo "=== Starting ${PROFILE_DURATION}s CPU profile capture in background ==="
curl -s -o "${PROFILE_OUTPUT}" "${PPROF_URL}" &
PROFILE_PID=$!
sleep 2  # let the profile start sampling

echo "=== Creating namespace ==="
kubectl apply -f "${REPRO_DIR}/00-namespace.yaml"

echo "=== Creating 50 decoy NodePools ==="
kubectl apply -f "${REPRO_DIR}/01-decoy-nodepools.yaml"

echo "=== Creating victim NodePool ==="
kubectl apply -f - <<'VICTIM_NP'
apiVersion: karpenter.sh/v1
kind: NodePool
metadata:
  name: repro-victim
  labels:
    purpose: karpenter-repro
spec:
  template:
    metadata:
      labels:
        node-role: repro-victim
    spec:
      expireAfter: 24h
      nodeClassRef:
        group: karpenter.k8s.aws
        kind: EC2NodeClass
        name: default
      requirements:
      - key: kubernetes.io/arch
        operator: In
        values: ["amd64"]
      - key: karpenter.sh/capacity-type
        operator: In
        values: ["on-demand"]
      - key: node.kubernetes.io/instance-type
        operator: In
        values: ["m5.large"]
VICTIM_NP

echo "=== Creating 3000 hog pods (cpu: 2000, ScheduleAnyway TSC on hostname) ==="
kubectl apply -f "${REPRO_DIR}/04-hog-pods.yaml"

echo "=== Creating 5 victim pods (100m CPU, 128Mi) ==="
kubectl apply -f - <<'VICTIM_PODS'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: repro-victim
  namespace: karpenter-repro
spec:
  replicas: 5
  selector:
    matchLabels:
      app: repro-victim
  template:
    metadata:
      labels:
        app: repro-victim
    spec:
      terminationGracePeriodSeconds: 0
      nodeSelector:
        node-role: repro-victim
      containers:
      - name: pause
        image: registry.k8s.io/pause:3.9
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
VICTIM_PODS

echo "=== Resources created. Waiting for profile to complete (${PROFILE_DURATION}s total) ==="
echo "=== Monitor with: kubectl -n kube-system logs -f <leader-pod> | grep 'scheduling simulation timed out' ==="
wait $PROFILE_PID
echo "=== Profile saved to ${PROFILE_OUTPUT} ==="
echo "=== Analyze with: go tool pprof -http=:9090 ${PROFILE_OUTPUT} ==="
