# Add the SQS and SSM VPC endpoints if we are creating a private cluster
# We need to grab all of the VPC details for the cluster in order to add the endpoint
# Add inbound rules for codeBuild security group, create temporary access entry
VPC_CONFIG=$(aws eks describe-cluster --name "$CLUSTER_NAME" --query "cluster.resourcesVpcConfig")
VPC_ID=$(echo $VPC_CONFIG | jq .vpcId -r)
SUBNET_IDS=($(echo $VPC_CONFIG | jq '.subnetIds | join(" ")' -r))
SHARED_NODE_SG=$((aws ec2 describe-security-groups --filters Name=tag:aws:cloudformation:stack-name,Values=eksctl-$CLUSTER_NAME-cluster Name=tag:aws:cloudformation:logical-id,Values=ClusterSharedNodeSecurityGroup --query "SecurityGroups[0]") | jq .GroupId -r)
EKS_CLUSTER_SG=$((aws ec2 describe-security-groups --filters Name=tag:aws:eks:cluster-name,Values=$CLUSTER_NAME  --query "SecurityGroups[0]") | jq .GroupId -r)

for SERVICE in "com.amazonaws.$REGION.ssm" "com.amazonaws.$REGION.eks" "com.amazonaws.$REGION.sqs"; do
  aws ec2 create-vpc-endpoint \
    --vpc-id "${VPC_ID}" \
    --vpc-endpoint-type Interface \
    --service-name "${SERVICE}" \
    --subnet-ids ${SUBNET_IDS[@]} \
    --security-group-ids ${EKS_CLUSTER_SG} \
    --tag-specifications 'ResourceType=vpc-endpoint,Tags=[{Key=testing/type,Value=e2e},{Key=testing/cluster,Value=$CLUSTER_NAME},{Key=github.com/run-url,Value=https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }}},{Key=karpenter.sh/discovery,Value=$CLUSTER_NAME}]'
done

aws ec2 authorize-security-group-ingress --group-id ${SHARED_NODE_SG} --protocol  all --source-group sg-0377e6d6574402945
aws ec2 authorize-security-group-ingress --group-id ${EKS_CLUSTER_SG} --protocol  all --source-group sg-0377e6d6574402945

# There is currently no VPC private endpoint for the IAM API. Therefore, we need to
# provision and manage an instance profile manually.
aws iam create-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}"
aws iam add-role-to-instance-profile --instance-profile-name "KarpenterNodeInstanceProfile-${CLUSTER_NAME}" --role-name "KarpenterNodeRole-${CLUSTER_NAME}"