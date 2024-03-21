aws ec2 delete-vpc-peering-connection --vpc-peering-connection-id "${VPC_PEERING_CONNECTION_ID}"
aws iam remove-role-from-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}" --role-name "KarpenterNodeRole-${CLUSTER_NAME}"
aws iam delete-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}"
aws iam delete-role-policy --role-name "${NODE_ROLE}" --policy-name "PullThroughCachePolicy"