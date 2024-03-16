aws iam remove-role-from-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}" --role-name "KarpenterNodeRole-${CLUSTER_NAME}"
aws iam delete-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}"
MANAGED_NG=$(aws eks list-nodegroups --cluster-name "${CLUSTER_NAME}" --query nodegroups --output text)
NODE_ROLE=$(aws eks describe-nodegroup --cluster-name "${CLUSTER_NAME}" --nodegroup-name "${MANAGED_NG}" --query nodegroup.nodeRole --output text | cut -d '/' -f 2)
aws iam delete-role-policy --role-name "${NODE_ROLE}" --policy-name "PullThroughCachePolicy"