// Copyright 2019-2024 The Liqo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resources

import (
	"context"
	"fmt"
	"sort"

	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sharingv1alpha1 "github.com/liqotech/liqo/apis/sharing/v1alpha1"
	"github.com/liqotech/liqo/pkg/utils/getters"
	liqolabels "github.com/liqotech/liqo/pkg/utils/labels"
	"github.com/liqotech/liqo/pkg/utils/slice"
)

// WellKnownResources contains the well known resources.
var WellKnownResources = []string{
	corev1.ResourceCPU.String(),
	corev1.ResourceMemory.String(),
	corev1.ResourceEphemeralStorage.String(),
	corev1.ResourcePods.String(),
}

// CPU returns the CPU quantity as a string.
func CPU(r corev1.ResourceList) string {
	result := r.Cpu().ScaledValue(resource.Milli)
	return fmt.Sprintf("%dm", result)
}

// Memory returns the memory quantity as a string.
func Memory(r corev1.ResourceList) string {
	result := float64(r.Memory().ScaledValue(resource.Mega)) / 1024
	return fmt.Sprintf("%.2fGiB", result)
}

// Pods returns the pods quantity as a string.
func Pods(r corev1.ResourceList) string {
	return r.Pods().String()
}

// EphemeralStorage returns the storage quantity as a string.
func EphemeralStorage(r corev1.ResourceList) string {
	result := float64(r.StorageEphemeral().ScaledValue(resource.Mega)) / 1024
	return fmt.Sprintf("%.2fGiB", result)
}

// Others returns the resources that are not well known.
func Others(r corev1.ResourceList) map[string]string {
	result := map[string]string{}

	keys := maps.Keys(r)
	sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys {
		if v, ok := (r)[k]; !slice.ContainsString(WellKnownResources, k.String()) && ok && v.Value() != 0 {
			result[k.String()] = v.String()
		}
	}
	return result
}

// GetAcquiredTotal returns the total acquired resources for a given cluster.
func GetAcquiredTotal(ctx context.Context, cl client.Client, clusterID string) (corev1.ResourceList, error) {
	rl, err := getters.ListResourceOfferByLabel(ctx, cl, metav1.NamespaceAll, liqolabels.RemoteLabelSelectorForCluster(clusterID))
	if err != nil {
		return corev1.ResourceList{}, err
	}
	return SumResourceOffers(rl), nil
}

// GetSharedTotal returns the total shared resources for a given cluster.
func GetSharedTotal(ctx context.Context, cl client.Client, clusterID string) (corev1.ResourceList, error) {
	rl, err := getters.ListResourceOfferByLabel(ctx, cl, metav1.NamespaceAll, liqolabels.LocalLabelSelectorForCluster(clusterID))
	if err != nil {
		return corev1.ResourceList{}, err
	}
	return SumResourceOffers(rl), nil
}

// SumResourceOffers sums the resources of a list of resource offers.
func SumResourceOffers(resourceoffers *sharingv1alpha1.ResourceOfferList) corev1.ResourceList {
	tot := corev1.ResourceList{}
	for i := range resourceoffers.Items {
		h := resourceoffers.Items[i].Spec.ResourceQuota.Hard
		if cpu, ok := tot[corev1.ResourceCPU]; !ok {
			tot[corev1.ResourceCPU] = h.Cpu().DeepCopy()
		} else {
			cpu.Add(h.Cpu().DeepCopy())
			tot[corev1.ResourceCPU] = cpu.DeepCopy()
		}

		if mem, ok := tot[corev1.ResourceMemory]; !ok {
			tot[corev1.ResourceMemory] = h.Memory().DeepCopy()
		} else {
			mem.Add(h.Memory().DeepCopy())
			tot[corev1.ResourceMemory] = mem.DeepCopy()
		}

		if storage, ok := tot[corev1.ResourceEphemeralStorage]; !ok {
			tot[corev1.ResourceEphemeralStorage] = h.StorageEphemeral().DeepCopy()
		} else {
			storage.Add(h.StorageEphemeral().DeepCopy())
			tot[corev1.ResourceEphemeralStorage] = storage.DeepCopy()
		}

		if pods, ok := tot[corev1.ResourcePods]; !ok {
			tot[corev1.ResourcePods] = h.Pods().DeepCopy()
		} else {
			pods.Add(h.Pods().DeepCopy())
			tot[corev1.ResourcePods] = pods.DeepCopy()
		}

		for k := range Others(h) {
			fmt.Println(k)
			q := h[corev1.ResourceName(k)]
			if t, ok := tot[corev1.ResourceName(k)]; !ok {
				tot[corev1.ResourceName(k)] = q.DeepCopy()
			} else {
				t.Add(q.DeepCopy())
				tot[corev1.ResourceName(k)] = t.DeepCopy()
			}
		}
	}
	return tot
}
