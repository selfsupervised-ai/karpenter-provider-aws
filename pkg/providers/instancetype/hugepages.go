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

package instancetype

import (
	"fmt"
	"strings"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/karpenter/pkg/utils/env"
	"sigs.k8s.io/karpenter/pkg/utils/resources"
)

const hugepage2MiByteSize int64 = 2 * 1024 * 1024

// Threaded from var.client_hugepages_percentage in
// iac/provider-aws/eks-cluster/main.tf so node-userdata.sh and the controller
// stay in lockstep at boot.
var hugepagesPercent = readHugepagesPercent()

// hugepagesEligibleFamilies are the Firecracker-eligible families consumed by
// the c8i-firecracker EC2NodeClass in
// iac/provider-aws/eks-cluster/karpenter-nodepools.tf. Restricting to these
// avoids advertising capacity on types that node-userdata.sh won't configure.
var hugepagesEligibleFamilies = map[string]struct{}{
	"c8i": {},
	"m8i": {},
	"r8i": {},
}

// zeroHugepages2Mi is returned for every non-eligible instance type. The
// caller in computeCapacity dereferences the pointer, so a shared singleton
// avoids ~795 throwaway heap allocations per cache refresh.
var zeroHugepages2Mi = resource.NewQuantity(0, resource.BinarySI)

var resourceHugePages2Mi = corev1.ResourceName(corev1.ResourceHugePagesPrefix + "2Mi")

func readHugepagesPercent() int64 {
	v := env.WithDefaultInt64("KARPENTER_HUGEPAGES_PERCENT", 0)
	if v < 0 || v > 100 {
		return 0
	}
	return v
}

func hugepages2Mi(info ec2types.InstanceTypeInfo) *resource.Quantity {
	if hugepagesPercent <= 0 || info.MemoryInfo == nil || info.MemoryInfo.SizeInMiB == nil {
		return zeroHugepages2Mi
	}
	family := string(info.InstanceType)
	if i := strings.IndexByte(family, '.'); i >= 0 {
		family = family[:i]
	}
	if _, ok := hugepagesEligibleFamilies[family]; !ok {
		return zeroHugepages2Mi
	}
	pageCount := (*info.MemoryInfo.SizeInMiB * 1024 * 1024 * hugepagesPercent / 100) / hugepage2MiByteSize
	return resources.Quantity(fmt.Sprintf("%dMi", pageCount*2))
}
