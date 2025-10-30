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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	v1 "github.com/aws/karpenter-provider-aws/pkg/apis/v1"
	awscache "github.com/aws/karpenter-provider-aws/pkg/cache"
	"github.com/aws/karpenter-provider-aws/pkg/operator/options"
	"github.com/aws/karpenter-provider-aws/pkg/providers/instancetype"
	"github.com/aws/karpenter-provider-aws/pkg/providers/pricing"
	"github.com/aws/karpenter-provider-aws/pkg/providers/subnet"
	"github.com/aws/karpenter-provider-aws/pkg/test"
	"github.com/patrickmn/go-cache"
	"github.com/samber/lo"
	"sigs.k8s.io/karpenter/pkg/cloudprovider"
	coreoptions "sigs.k8s.io/karpenter/pkg/operator/options"
	coretest "sigs.k8s.io/karpenter/pkg/test"
)

func main() {
	ctx := coreoptions.ToContext(context.Background(), coretest.Options(coretest.OptionsFields{
		FeatureGates: coretest.FeatureGates{ReservedCapacity: lo.ToPtr(false)},
	}))
	ctx = options.ToContext(ctx, test.Options(test.OptionsFields{
		ClusterName:     lo.ToPtr("docs-gen"),
		ClusterEndpoint: lo.ToPtr("https://docs-gen.aws"),
	}))
	cfg := lo.Must(config.LoadDefaultConfig(ctx))
	ec2api := ec2.NewFromConfig(cfg)
	subnetProvider := subnet.NewDefaultProvider(ec2api, cache.New(awscache.DefaultTTL, awscache.DefaultCleanupInterval), cache.New(awscache.AvailableIPAddressTTL, awscache.DefaultCleanupInterval), cache.New(awscache.AssociatePublicIPAddressTTL, awscache.DefaultCleanupInterval))
	instanceTypeProvider := instancetype.NewDefaultProvider(
		cache.New(awscache.InstanceTypesZonesAndOfferingsTTL, awscache.DefaultCleanupInterval),
		cache.New(awscache.InstanceTypesZonesAndOfferingsTTL, awscache.DefaultCleanupInterval),
		cache.New(awscache.DiscoveredCapacityCacheTTL, awscache.DefaultCleanupInterval),
		ec2api,
		subnetProvider,
		pricing.NewDefaultProvider(
			pricing.NewAPI(cfg),
			ec2api,
			cfg.Region,
			true,
		),
		nil,
		awscache.NewUnavailableOfferings(),
		instancetype.NewDefaultResolver(
			cfg.Region,
		),
	)
	if err := instanceTypeProvider.UpdateInstanceTypes(ctx); err != nil {
		log.Fatalf("updating instance types, %s", err)
	}
	if err := instanceTypeProvider.UpdateInstanceTypeOfferings(ctx); err != nil {
		log.Fatalf("updating instance types offerings, %s", err)
	}
	// Fake a NodeClass so we can use it to get InstanceTypes
	nodeClass := &v1.EC2NodeClass{
		Spec: v1.EC2NodeClassSpec{
			AMISelectorTerms: []v1.AMISelectorTerm{{
				Alias: "al2023@latest",
			}},
			SubnetSelectorTerms: []v1.SubnetSelectorTerm{
				{
					Tags: map[string]string{
						"*": "*",
					},
				},
			},
		},
	}
	subnets, err := subnetProvider.List(ctx, nodeClass)
	if err != nil {
		log.Fatalf("listing subnets, %s", err)
	}
	nodeClass.Status.Subnets = lo.Map(subnets, func(ec2subnet ec2types.Subnet, _ int) v1.Subnet {
		return v1.Subnet{
			ID:   *ec2subnet.SubnetId,
			Zone: *ec2subnet.AvailabilityZone,
		}
	})
	instanceTypes, err := instanceTypeProvider.List(ctx, nodeClass)
	if err != nil {
		log.Fatalf("listing instance types, %s", err)
	}

	// Write to file
	filename := "kwok/cloudprovider/instance_types.json"
	err = WriteInstanceTypesToFile(instanceTypes, filename)
	if err != nil {
		log.Fatalf("Failed to write instance types to file: %v", err)
	}
	fmt.Printf("Successfully wrote %d instance types to %s\n", len(instanceTypes), filename)
}

type SerializableInstanceType struct {
	Name             string                 `json:"name"`
	Offerings        []SerializableOffering `json:"offerings"`
	Architecture     string                 `json:"architecture"`
	OperatingSystems []string               `json:"operatingSystems"`
	Resources        map[string]string      `json:"resources"`
}

type SerializableOffering struct {
	Price        float64                   `json:"Price"`
	Available    bool                      `json:"Available"`
	Requirements []SerializableRequirement `json:"Requirements"`
}

type SerializableRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

func WriteInstanceTypesToFile(instanceTypes []*cloudprovider.InstanceType, filename string) error {
	serializable := make([]SerializableInstanceType, len(instanceTypes))
	for i, it := range instanceTypes {
		archValues := it.Requirements.Get("kubernetes.io/arch").Values()
		arch := "amd64"
		if len(archValues) > 0 {
			arch = archValues[0]
		}
		osValues := it.Requirements.Get("kubernetes.io/os").Values()
		if len(osValues) == 0 {
			osValues = []string{"linux"}
		}

		serializable[i] = SerializableInstanceType{
			Name:             it.Name,
			Offerings:        make([]SerializableOffering, len(it.Offerings)),
			Architecture:     arch,
			OperatingSystems: osValues,
			Resources:        make(map[string]string),
		}
		for k, v := range it.Capacity {
			serializable[i].Resources[string(k)] = v.String()
		}
		for j, offering := range it.Offerings {
			reqs := make([]SerializableRequirement, 0)
			for _, req := range offering.Requirements.Values() {
				values := req.Values()
				if len(values) == 0 && req.Operator() == "DoesNotExist" {
					values = []string{}
				} else if len(values) == 0 {
					continue
				}
				reqs = append(reqs, SerializableRequirement{
					Key:      req.Key,
					Operator: string(req.Operator()),
					Values:   values,
				})
			}
			serializable[i].Offerings[j] = SerializableOffering{
				Price:        offering.Price,
				Available:    offering.Available,
				Requirements: reqs,
			}
		}
	}
	data, err := json.MarshalIndent(serializable, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}
