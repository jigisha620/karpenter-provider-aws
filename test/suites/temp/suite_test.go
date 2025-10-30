/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package temp_test

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/karpenter/pkg/test"
	ct "sigs.k8s.io/karpenter/test/pkg/environment/common"

	v1 "github.com/aws/karpenter-provider-aws/pkg/apis/v1"
	environmentaws "github.com/aws/karpenter-provider-aws/test/pkg/environment/aws"
	"github.com/aws/karpenter-provider-aws/test/pkg/environment/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	karpv1 "sigs.k8s.io/karpenter/pkg/apis/v1"
)

// var env *ct.Environment

func TestTemp(t *testing.T) {
	RegisterFailHandler(Fail)
	BeforeSuite(func() {
		env = ct.NewEnvironment(t)
		e = environmentaws.NewEnvironment(t)
	})
	AfterSuite(func() {
		env.Stop()
		e.Stop()
	})
	RunSpecs(t, "NodeClaim")
}

var testLabels = map[string]string{
	test.DiscoveryLabel: "owned",
}
var labelSelector = labels.SelectorFromSet(testLabels)
var e *environmentaws.Environment
var env *ct.Environment
var nodeClass *v1.EC2NodeClass
var nodePool *karpv1.NodePool

var _ = BeforeEach(func() {
	env.BeforeEach()
	e.BeforeEach()
	nodeClass = e.DefaultEC2NodeClass()
	nodePool = e.DefaultNodePool(nodeClass)
})
var _ = AfterEach(func() { env.Cleanup() })
var _ = AfterEach(func() { env.AfterEach() })

var _ = Describe("GarbageCollection", func() {
	FIt("should consolidate nodes after the workload is scaled down", func() {
		numPods := 2
		dep := test.Deployment(test.DeploymentOptions{
			Replicas: int32(numPods),
			PodOptions: test.PodOptions{
				ObjectMeta: metav1.ObjectMeta{
					Labels: testLabels,
				},
				ResourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("1"),
					},
				},
			}})
		// Hostname anti-affinity to require one pod on each node
		dep.Spec.Template.Spec.Affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						LabelSelector: dep.Spec.Selector,
						TopologyKey:   corev1.LabelHostname,
					},
				},
			},
		}
		env.ExpectCreated(nodeClass, nodePool, dep)

		env.EventuallyExpectCreatedNodeClaimCount("==", numPods)
		nodes := env.EventuallyExpectCreatedNodeCount("==", numPods)
		cost := env.GetClusterCost()
		Expect(cost).To(BeNumerically(">", 0.0))
		By(fmt.Sprintf("cluster cost is: %v", cost))

		env.EventuallyExpectHealthyPodCount(labelSelector, numPods)

		By("adding finalizers to the nodes to prevent termination")
		for _, node := range nodes {
			Expect(env.Client.Get(env.Context, client.ObjectKeyFromObject(node), node)).To(Succeed())
			node.Finalizers = append(node.Finalizers, common.TestingFinalizer)
			env.ExpectUpdated(node)
		}

		dep.Spec.Replicas = lo.ToPtr[int32](1)
		By("making the nodes empty")
		// Update the deployment to only contain 1 replica.
		env.ExpectUpdated(dep)

		env.ConsistentlyExpectDisruptionsUntilNoneLeft(numPods, numPods, 2*time.Minute)
	})
})
