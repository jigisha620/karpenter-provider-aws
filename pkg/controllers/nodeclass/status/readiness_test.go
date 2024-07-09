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

package status_test

import (
	"github.com/awslabs/operatorpkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"github.com/aws/karpenter-provider-aws/pkg/apis/v1beta1"
	"github.com/aws/karpenter-provider-aws/pkg/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "sigs.k8s.io/karpenter/pkg/test/expectations"
)

var _ = Describe("NodeClass Status Condition Controller", func() {
	BeforeEach(func() {
		nodeClass = test.EC2NodeClass(v1beta1.EC2NodeClass{
			Spec: v1beta1.EC2NodeClassSpec{
				SubnetSelectorTerms: []v1beta1.SubnetSelectorTerm{
					{
						Tags: map[string]string{"*": "*"},
					},
				},
				SecurityGroupSelectorTerms: []v1beta1.SecurityGroupSelectorTerm{
					{
						Tags: map[string]string{"*": "*"},
					},
				},
				AMISelectorTerms: []v1beta1.AMISelectorTerm{
					{
						Tags: map[string]string{"*": "*"},
					},
				},
			},
		})
	})
	It("should update status condition on nodeClass as Ready", func() {
		ExpectApplied(ctx, env.Client, nodeClass)
		ExpectObjectReconciled(ctx, env.Client, statusController, nodeClass)
		nodeClass = ExpectExists(ctx, env.Client, nodeClass)
		Expect(nodeClass.Status.Conditions).To(HaveLen(5))
		Expect(nodeClass.StatusConditions().Get(status.ConditionReady).IsTrue()).To(BeTrue())
	})
	It("should update status condition as Not Ready", func() {
		nodeClass.Spec.SecurityGroupSelectorTerms = []v1beta1.SecurityGroupSelectorTerm{
			{
				Tags: map[string]string{"foo": "invalid"},
			},
		}
		ExpectApplied(ctx, env.Client, nodeClass)
		ExpectObjectReconciled(ctx, env.Client, statusController, nodeClass)
		nodeClass = ExpectExists(ctx, env.Client, nodeClass)

		Expect(nodeClass.StatusConditions().Get(status.ConditionReady).IsFalse()).To(BeTrue())
		Expect(nodeClass.StatusConditions().Get(status.ConditionReady).Message).To(Equal("SecurityGroupsReady=False"))
	})
	It("should update status condition as Not Ready if EC2nodeClass is terminating", func() {
		nodeClass.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		ExpectApplied(ctx, env.Client, nodeClass)
		ExpectObjectReconciled(ctx, env.Client, statusController, nodeClass)
		nodeClass = ExpectExists(ctx, env.Client, nodeClass)

		Expect(nodeClass.StatusConditions().Get(status.ConditionReady).IsFalse()).To(BeTrue())
		Expect(nodeClass.StatusConditions().Get(status.ConditionReady).Message).To(Equal("NodeClass is Terminating"))
	})
})
