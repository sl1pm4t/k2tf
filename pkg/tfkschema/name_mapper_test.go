package tfkschema

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestToTerraformAttributeName(t *testing.T) {
	type args struct {
		field *reflect.StructField
		path  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"replicas",
			args{
				&reflect.StructField{
					Name: "Replicas",
					Tag:  `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`,
				},
				"",
			},
			"replicas",
		},
		{
			"non_resource_urls",
			args{
				&reflect.StructField{
					Name: "NonResourceURLs",
					Tag:  `json:"nonResourceURLs,omitempty" protobuf:"bytes,5,rep,name=nonResourceURLs"`,
				},
				"kubernetes_cluster_role.rule",
			},
			"non_resource_urls",
		},
		{
			"non_resource_urls",
			args{
				&reflect.StructField{
					Name: "NonResourceURLs",
					Tag:  `json:"nonResourceURLs,omitempty" protobuf:"bytes,5,rep,name=nonResourceURLs"`,
				},
				"kubernetes_role.rule",
			},
			"non_resource_urls",
		},
		{
			"external_ips",
			args{
				&reflect.StructField{
					Name: "ExternalIPs",
					Tag:  `json:"externalIPs,omitempty" protobuf:"bytes,1,rep,name=externalIPs"`,
				},
				"kubernetes_service.spec",
			},
			"external_ips",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToTerraformAttributeName(tt.args.field, tt.args.path); got != tt.want {
				t.Errorf("ToTerraformAttributeName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToTerraformSubBlockName(t *testing.T) {
	type args struct {
		field *reflect.StructField
		path  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"container",
			args{
				&reflect.StructField{
					Name: "Container",
					Tag:  `json:"containers" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=containers"`,
				},
				"",
			},
			"container",
		},
		{
			"port",
			args{
				&reflect.StructField{
					Name: "ContainerPort",
					Tag:  `json:"ports,omitempty" patchStrategy:"merge" patchMergeKey:"containerPort" protobuf:"bytes,6,rep,name=ports"`,
				},
				"",
			},
			"port",
		},
		{
			"match_labels",
			args{
				&reflect.StructField{
					Name: "MatchLabels",
					Tag:  `json:"matchLabels,omitempty" protobuf:"bytes,1,rep,name=matchLabels"`,
				},
				"",
			},
			"match_labels",
		},
		{
			"update_strategy/daemonset",
			args{
				&reflect.StructField{
					Name: "UpdateStrategy",
					Tag:  `json:"updateStrategy,omitempty" protobuf:"bytes,1,rep,name=updateStrategy"`,
				},
				"daemonset.spec.",
			},
			"strategy",
		},
		{
			"update_strategy/statefulset",
			args{
				&reflect.StructField{
					Name: "UpdateStrategy",
					Tag:  `json:"updateStrategy,omitempty" protobuf:"bytes,1,rep,name=updateStrategy"`,
				},
				"stateful_set.spec.",
			},
			"update_strategy",
		},
		{
			"volume_source",
			args{
				&reflect.StructField{
					Name: "VolumeSource",
					Tag:  `json:",inline" protobuf:"bytes,2,opt,name=volumeSource"`,
				},
				"",
			},
			"volume_source",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToTerraformSubBlockName(tt.args.field, tt.args.path); got != tt.want {
				t.Errorf("ToTerraformSubBlockName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_NormalizeTerraformName(t *testing.T) {
	type args struct {
		s          string
		toSingular bool
		path       string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"labels",
			args{
				"labels",
				true,
				"",
			},
			"labels",
		},
		{
			"match_labels",
			args{
				"matchLabels",
				true,
				"",
			},
			"match_labels",
		},
		{
			"metadata",
			args{
				"metadata",
				true,
				"",
			},
			"metadata",
		},
		{
			"non_resource_urls",
			args{
				"nonResourceURLs",
				false,
				"cluster_role.rule.",
			},
			"non_resource_urls",
		},
		{
			"non_resource_urls",
			args{
				"nonResourceURLs",
				false,
				"role.rule.",
			},
			"non_resource_urls",
		},
		{
			"LimitRangeSpec.Limits",
			args{
				"limits",
				true,
				"limit_range.spec.",
			},
			"limit",
		},
		{
			"image_pull_secrets",
			args{
				"imagePullSecrets",
				true,
				"",
			},
			"image_pull_secrets",
		},
		{
			"capabilities",
			args{
				"capabilities",
				true,
				"",
			},
			"capabilities",
		},
		{
			"metallb-system:speaker",
			args{
				"metallb-system:speaker",
				true,
				"",
			},
			"metallb_system_speaker",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeTerraformName(tt.args.s, tt.args.toSingular, tt.args.path); got != tt.want {
				t.Errorf("normalizeTerraformName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToTerraformResourceType(t *testing.T) {
	type args struct {
		obj runtime.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Pod",
			args{
				obj: &v1.Pod{
					TypeMeta: metav1.TypeMeta{
						Kind: "Pod",
					},
				},
			},
			"kubernetes_pod",
		},
		{
			"DaemonSet",
			args{
				obj: &appsv1.DaemonSet{
					TypeMeta: metav1.TypeMeta{
						Kind: "DaemonSet",
					},
				},
			},
			"kubernetes_daemonset",
		},
		{
			"Deployment",
			args{
				obj: &appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind: "Deployment",
					},
				},
			},
			"kubernetes_deployment",
		},
		{
			"Service",
			args{
				obj: &v1.Service{
					TypeMeta: metav1.TypeMeta{
						Kind: "Service",
					},
				},
			},
			"kubernetes_service",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToTerraformResourceType(tt.args.obj); got != tt.want {
				t.Errorf("ToTerraformResourceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeTerraformMapKey(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"slash", args{"kubernetes/foo"}, `"kubernetes/foo"`},
		{"dot", args{"kubernetes.io"}, `"kubernetes.io"`},
		{"no_change", args{"kubernetes"}, `kubernetes`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeTerraformMapKey(tt.args.s); got != tt.want {
				t.Errorf("NormalizeTerraformMapKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
