aws iam remove-role-from-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}" --role-name "KarpenterNodeRole-${CLUSTER_NAME}"
aws iam delete-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}"
MANAGED_NG=$(aws eks list-nodegroups --cluster-name "${CLUSTER_NAME}" --query nodegroups --output text)
NODE_ROLE=$(aws eks describe-nodegroup --cluster-name "${CLUSTER_NAME}" --nodegroup-name "${MANAGED_NG}" --query nodegroup.nodeRole --output text | cut -d '/' -f 2)
aws iam delete-role-policy --role-name "${NODE_ROLE}" --policy-name "PullThroughCachePolicy"

VPC_CONFIG=$(aws eks describe-cluster --name "$CLUSTER_NAME" --query "cluster.resourcesVpcConfig")
VPC_ID=$(echo $VPC_CONFIG | jq .vpcId -r)

for SERVICE in "com.amazonaws.$REGION.ssm" "com.amazonaws.$REGION.eks" "com.amazonaws.$REGION.sqs"; do
VPC_ENDPOINT_ID=$((aws ec2 describe-vpc-endpoints --filters Name=vpc-id,Values=$VPC_ID Name=service-name,Values=$SERVICE --query "VpcEndpoints[0]") | jq .VpcEndpointId -r)
aws ec2 modify-vpc-endpoint --vpc-endpoint-id $VPC_ENDPOINT_ID --add-security-group-ids $CODEBUILD_SG
aws ec2 modify-vpc-endpoint --vpc-endpoint-id $VPC_ENDPOINT_ID --remove-security-group-ids $EKS_CLUSTER_SG
done

for SERVICE in "com.amazonaws.$REGION.ecr.api" "com.amazonaws.$REGION.logs" "com.amazonaws.$REGION.sts" "com.amazonaws.$REGION.ecr.dkr" "com.amazonaws.$REGION.ec2"; do
VPC_ENDPOINT_ID=$((aws ec2 describe-vpc-endpoints --filters Name=vpc-id,Values=$VPC_ID Name=service-name,Values=$SERVICE --query "VpcEndpoints[0]") | jq .VpcEndpointId -r)
aws ec2 modify-vpc-endpoint --vpc-endpoint-id $VPC_ENDPOINT_ID --add-security-group-ids $EKS_CLUSTER_SG
aws ec2 modify-vpc-endpoint --vpc-endpoint-id $VPC_ENDPOINT_ID --remove-security-group-ids $SHARED_NODE_SG
done