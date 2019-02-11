package main

import (
	"fmt"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func WriteObject(obj runtime.Object, dst *hclwrite.Body) {
	switch obj.GetObjectKind().GroupVersionKind().Kind {
	case "ConfigMap":
		WriteConfigMap(obj.(*corev1.ConfigMap), dst)
	case "Deployment":
		WriteDeployment(obj.(*v1.Deployment), dst)
	}
}

func encodeMetadataBlock(meta *metav1.ObjectMeta) *hclwrite.Block {
	blk := hclwrite.NewBlock("metadata", nil)
	blk.Body().SetAttributeValue("name", convertCtyValue(meta.Name))

	if meta.Namespace != "" {
		blk.Body().SetAttributeValue("namespace", convertCtyValue(meta.Namespace))
	}

	if len(meta.Labels) > 0 {
		lbls := hclwrite.NewBlock("labels", nil)
		encodeMap(meta.Labels, lbls.Body())
		blk.Body().AppendBlock(lbls)
	}

	if len(meta.Annotations) > 0 {
		anno := hclwrite.NewBlock("annotations", nil)
		encodeMap(meta.Annotations, anno.Body())
		blk.Body().AppendBlock(anno)
	}
	return blk
}

func WriteConfigMap(obj *corev1.ConfigMap, dst *hclwrite.Body) {
	root := hclwrite.NewBlock("resource", []string{"kubernetes_config_map", obj.Name})
	dst.AppendBlock(root)

	root.Body().AppendBlock(encodeMetadataBlock(&obj.ObjectMeta))

	if len(obj.Data) > 0 {
		data := hclwrite.NewBlock("data", nil)
		encodeMap(obj.Data, data.Body())
		root.Body().AppendBlock(data)
	}

}

func WriteDeployment(obj *v1.Deployment, dst *hclwrite.Body) {
	root := hclwrite.NewBlock("resource", []string{"kubernetes_deployment", obj.Name})
	dst.AppendBlock(root)

	root.Body().AppendBlock(encodeMetadataBlock(&obj.ObjectMeta))

	spec := hclwrite.NewBlock("spec", nil)
	spec.Body().SetAttributeValue("replicas", convertCtyValue(obj.Spec.Replicas))

	root.Body().AppendBlock(spec)
}

func convertCtyValue(val interface{}) cty.Value {
	switch val.(type) {
	case string:
		return cty.StringVal(val.(string))
	case bool:
		return cty.BoolVal(val.(bool))
	case int:
		return cty.NumberIntVal(int64(val.(int)))
	case int32:
		return cty.NumberIntVal(int64(val.(int32)))
	case *int32:
		val = *val.(*int32)
		return cty.NumberIntVal(int64(val.(int32)))
	case int64:
		return cty.NumberIntVal(int64(val.(int64)))
	case float64:
		return cty.NumberFloatVal(float64(val.(float64)))
	case map[string]interface{}:
		ctyMap := map[string]cty.Value{}
		for k, v := range val.(map[string]interface{}) {
			ctyMap[k] = convertCtyValue(v)
		}

		return cty.ObjectVal(ctyMap)
	default:
		fmt.Printf("[!] WARN: unhandled variable type: %T \n", val)
	}
	return cty.NilVal
}

func encodeMap(m map[string]string, dst *hclwrite.Body) {
	for k, v := range m {
		dst.SetAttributeValue(k, convertCtyValue(v))
	}
}
