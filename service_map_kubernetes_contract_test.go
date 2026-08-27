package flashduty

import (
	"encoding/json"
	"testing"
)

func TestServiceMapKubernetesContractDecodesTopologyAndSummary(t *testing.T) {
	topologyPayload := []byte(`{
		"coverage": {
			"kubernetes_placements_returned": 1,
			"kubernetes_placements_omitted": 0,
			"kubernetes_projection_status": "complete",
			"kubernetes_canonical_count": 1
		},
		"nodes": [{
			"host_id": "host-a",
			"id": "entity-a",
			"kind": "container_workload",
			"display_name": "orders",
			"kubernetes_placements": [{
				"status": "canonical",
				"resolution_status": "canonical",
				"instance_id": "container-a",
				"pod_ref": {"uid": "pod-a", "namespace": "shop"},
				"canonical": {
					"canonical_cluster_id": "cluster-a",
					"context_revision": "revision-a",
					"source_observed_at_ms": 1787730000000,
					"pod_ref": {"uid": "pod-a", "namespace": "shop", "name": "orders-1"},
					"workload": {"kind": "Deployment", "uid": "workload-a", "name": "orders"},
					"services": [{"namespace": "shop", "name": "orders", "uid": "service-a"}]
				},
				"source": "cgroup",
				"confidence": "high"
			}]
		}]
	}`)

	var topology ServiceMapTopologyResponse
	if err := json.Unmarshal(topologyPayload, &topology); err != nil {
		t.Fatalf("unmarshal topology: %v", err)
	}
	if topology.Coverage.KubernetesProjectionStatus != "complete" || topology.Coverage.KubernetesCanonicalCount != 1 {
		t.Fatalf("coverage = %+v", topology.Coverage)
	}
	if len(topology.Nodes) != 1 || len(topology.Nodes[0].KubernetesPlacements) != 1 {
		t.Fatalf("nodes = %+v", topology.Nodes)
	}
	placement := topology.Nodes[0].KubernetesPlacements[0]
	if placement.Canonical == nil || placement.Canonical.CanonicalClusterID != "cluster-a" {
		t.Fatalf("canonical placement = %+v", placement.Canonical)
	}
	if placement.Canonical.Workload == nil || placement.Canonical.Workload.Name != "orders" {
		t.Fatalf("canonical workload = %+v", placement.Canonical.Workload)
	}
	if len(placement.Canonical.Services) != 1 || placement.Canonical.Services[0].Name != "orders" {
		t.Fatalf("canonical services = %+v", placement.Canonical.Services)
	}

	summaryPayload := []byte(`{
		"kubernetes_placements_omitted": 2,
		"kubernetes_placements": [{
			"host_id": "host-a",
			"entity_id": "entity-a",
			"status": "canonical",
			"pod_uid": "pod-a",
			"canonical": {
				"canonical_cluster_id": "cluster-a",
				"context_revision": "revision-a",
				"source_observed_at_ms": 1787730000000,
				"pod_ref": {"uid": "pod-a"}
			}
		}]
	}`)
	var summary ServiceMapSummaryResponse
	if err := json.Unmarshal(summaryPayload, &summary); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if summary.KubernetesPlacementsOmitted != 2 || len(summary.KubernetesPlacements) != 1 ||
		summary.KubernetesPlacements[0].Canonical == nil {
		t.Fatalf("summary placements = %+v omitted=%d", summary.KubernetesPlacements, summary.KubernetesPlacementsOmitted)
	}
}
