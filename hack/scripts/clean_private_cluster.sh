#!/usr/bin/env bash

# Delete route table entry for cluster
subnet_config=$(aws ec2 describe-subnets --filters Name=vpc-id,Values="${VPC_CB}" Name=tag:aws-cdk:subnet-type,Values=Private --query "Subnets")
echo "$subnet_config" | jq '.[].SubnetId' -r |
while read -r subnet;
do
  ROUTE_TABLE_ID=$((aws ec2 describe-route-tables --filters Name=vpc-id,Values="${VPC_CB}" Name=association.subnet-id,Values="$subnet" --query "RouteTables[0].RouteTableId") | jq -r)
  aws ec2 delete-route --route-table-id "$ROUTE_TABLE_ID" --destination-cidr-block "${CIDR_BLOCK}"
done

# Delete VPC peering connection
aws ec2 delete-vpc-peering-connection --vpc-peering-connection-id "${VPC_PEERING_CONNECTION_ID}"

# Delete instance profile
aws iam remove-role-from-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}" --role-name "KarpenterNodeRole-${CLUSTER_NAME}"
aws iam delete-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}"

# Delete private registry policy for pull through cache
aws iam delete-role-policy --role-name "${NODE_ROLE}" --policy-name "PullThroughCachePolicy"

# Delete cluster
eksctl delete cluster --name "${CLUSTER_NAME}" --timeout 60m --wait || true
