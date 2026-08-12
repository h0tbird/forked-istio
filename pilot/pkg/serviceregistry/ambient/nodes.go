// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ambient

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"istio.io/api/label"
	"istio.io/istio/pilot/pkg/util/protoconv"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/kube/multicluster"
	"istio.io/istio/pkg/log"
	"istio.io/istio/pkg/ptr"
	"istio.io/istio/pkg/slices"
	"istio.io/istio/pkg/workloadapi"
)

type Node struct {
	Name     string
	Locality *workloadapi.Locality
}

func (n Node) ResourceName() string {
	return n.Name
}

func (n Node) Equals(o Node) bool {
	return n.Name == o.Name &&
		protoconv.Equals(n.Locality, o.Locality)
}

func GlobalNodesCollection(
	ctrl *multicluster.Controller,
	nodes krt.Collection[krt.Collection[krt.ObjectWithCluster[*v1.Node]]],
	opts ...krt.CollectionOption,
) krt.Collection[krt.Collection[krt.ObjectWithCluster[Node]]] {
	return krt.NewCollection(
		nodes,
		func(ctx krt.HandlerContext, col krt.Collection[krt.ObjectWithCluster[*v1.Node]]) *krt.Collection[krt.ObjectWithCluster[Node]] {
			clusterID := col.Metadata()[multicluster.ClusterKRTMetadataKey]
			if clusterID == nil {
				panic("cluster metadata is nil for Node collection")
			}
			id, ok := clusterID.(cluster.ID)
			if !ok {
				panic(fmt.Sprintf("invalid cluster metadata set on Node collection: %v", clusterID))
			}
			// N.B the inner collection must be shut down with the cluster it belongs to, NEVER with the
			// top-level stop. Otherwise it outlives the cluster and keeps the cluster's client, informers
			// and caches alive: a rebuilt cluster (for instance on a token rotation) leaks all of it.
			c := krt.FetchOne(ctx, ctrl.Clusters(), krt.FilterKey(id.String()))
			if c == nil {
				log.Warnf("Cluster %s is gone, skipping node locality collection", id)
				return nil
			}
			innerOpts := append(slices.Clone(opts),
				krt.WithName(fmt.Sprintf("ambient/NodeLocalityWithCluster[%s]", id)),
				krt.WithStop((*c).GetStop()),
				krt.WithMetadata(krt.Metadata{
					multicluster.ClusterKRTMetadataKey: id,
				}),
			)
			nc := krt.NewCollection(col, func(ctx krt.HandlerContext, obj krt.ObjectWithCluster[*v1.Node]) *krt.ObjectWithCluster[Node] {
				k := ptr.Flatten(obj.Object)
				if k == nil {
					log.Warnf("Node %s is nil, skipping", obj.ClusterID)
					return nil
				}
				node := &Node{
					Name: k.Name,
				}
				region := k.GetLabels()[v1.LabelTopologyRegion]
				zone := k.GetLabels()[v1.LabelTopologyZone]
				subzone := k.GetLabels()[label.TopologySubzone.Name]

				if region != "" || zone != "" || subzone != "" {
					node.Locality = &workloadapi.Locality{
						Region:  region,
						Zone:    zone,
						Subzone: subzone,
					}
				}

				return &krt.ObjectWithCluster[Node]{
					ClusterID: obj.ClusterID,
					Object:    node,
				}
			}, innerOpts...)
			return ptr.Of(nc)
		},
		opts...)
}

// NodesCollection maps a node to it's locality.
// In many environments, nodes change frequently causing excessive recomputation of workloads.
// By making an intermediate collection we can reduce the times we need to trigger dependants (locality should ~never change).
func NodesCollection(nodes krt.Collection[*v1.Node], opts ...krt.CollectionOption) krt.Collection[Node] {
	return krt.NewCollection(nodes, func(ctx krt.HandlerContext, k *v1.Node) *Node {
		node := &Node{
			Name: k.Name,
		}
		region := k.GetLabels()[v1.LabelTopologyRegion]
		zone := k.GetLabels()[v1.LabelTopologyZone]
		subzone := k.GetLabels()[label.TopologySubzone.Name]

		if region != "" || zone != "" || subzone != "" {
			node.Locality = &workloadapi.Locality{
				Region:  region,
				Zone:    zone,
				Subzone: subzone,
			}
		}

		return node
	}, opts...)
}
