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
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func TestHugepages2Mi(t *testing.T) {
	// Literal byte counts — recomputing the production formula in the test
	// would mask off-by-one regressions, so these are hand-checked.
	cases := []struct {
		name     string
		instance ec2types.InstanceType
		memMiB   int64
		pct      int64
		expected int64
	}{
		{"c8i.2xlarge 80%", "c8i.2xlarge", 16384, 80, 13742637056},
		{"c8i.4xlarge 80%", "c8i.4xlarge", 32768, 80, 27487371264},
		{"c8i.2xlarge 50%", "c8i.2xlarge", 16384, 50, 8589934592},
		{"non-firecracker family", "t3.medium", 4096, 80, 0},
		{"feature disabled (percent zero)", "c8i.2xlarge", 16384, 0, 0},
	}

	originalPercent := hugepagesPercent
	t.Cleanup(func() { hugepagesPercent = originalPercent })

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hugepagesPercent = tc.pct
			info := ec2types.InstanceTypeInfo{
				InstanceType: tc.instance,
				MemoryInfo:   &ec2types.MemoryInfo{SizeInMiB: aws.Int64(tc.memMiB)},
			}
			if got := hugepages2Mi(info).Value(); got != tc.expected {
				t.Fatalf("got %d bytes, want %d bytes", got, tc.expected)
			}
		})
	}
}

func TestReadHugepagesPercent(t *testing.T) {
	cases := []struct {
		name     string
		envValue string
		expected int64
	}{
		{"valid 80", "80", 80},
		{"valid 0", "0", 0},
		{"valid 100", "100", 100},
		{"out of range high", "101", 0},
		{"out of range negative", "-1", 0},
		{"non-numeric", "abc", 0},
		{"empty string", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("KARPENTER_HUGEPAGES_PERCENT", tc.envValue)
			if got := readHugepagesPercent(); got != tc.expected {
				t.Fatalf("got %d, want %d", got, tc.expected)
			}
		})
	}
}

func TestHugepages2Mi_MissingMemoryInfo(t *testing.T) {
	originalPercent := hugepagesPercent
	t.Cleanup(func() { hugepagesPercent = originalPercent })
	hugepagesPercent = 80

	info := ec2types.InstanceTypeInfo{InstanceType: "c8i.2xlarge"}
	if got := hugepages2Mi(info).Value(); got != 0 {
		t.Fatalf("expected 0 bytes when MemoryInfo is nil, got %d", got)
	}
}
