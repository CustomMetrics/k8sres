package k8sresmetric

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spyzhov/ajson"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MetricsInfo struct {
	Data []*ajson.Node
	Path string
	Obj  string
	// Separate out Label keys and path
	// so that we do not have to iterate over map
	// and risk being inconsistent with key and values.
	LabelKeys []string
	LabelPath []string
}
type kMetrics struct {
	metricsMap map[string]*MetricsInfo
	crMap      map[string]unstructured.UnstructuredList
}

func (k *kMetrics) RegisterMetric(m MetricsConfig) error {
	k.metricsMap[m.Name] = &MetricsInfo{
		Obj:       m.Properties.Object,
		Path:      m.Properties.Value,
		LabelKeys: []string{},
		Data:      []*ajson.Node{},
	}

	for key, path := range m.Properties.Labels {
		k.metricsMap[m.Name].LabelKeys = append(k.metricsMap[m.Name].LabelKeys, key)
		k.metricsMap[m.Name].LabelPath = append(k.metricsMap[m.Name].LabelPath, path)
	}

	return nil
}

func (k *kMetrics) LabelNames(metric string) []string {
	return k.metricsMap[metric].LabelKeys
}

func (k *kMetrics) Update() error {
	var err error

	// Clear existing cr list.
	k.crMap = make(map[string]unstructured.UnstructuredList)
	for key, val := range k.metricsMap {
		v, _ := getGVK(val.Obj)

		// Check if the object related to this GVK has been scraped before.
		if _, ok := k.crMap[v.Kind]; !ok {
			u := unstructured.UnstructuredList{}
			u.SetGroupVersionKind(v)
			// TODO: add label selector.
			err = cl.List(context.Background(), &u, &client.ListOptions{Namespace: ""})
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			// Save the CRs in map.
			k.crMap[v.Kind] = u
		}

		var vals []*ajson.Node
		// Iterate over resources present in the unstructed list.
		for _, i := range k.crMap[v.Kind].Items {
			// Convert the resource into byte.
			b, err := i.MarshalJSON()
			if err != nil {
				return err
			}
			// Unmarshall the byte to *ajson.Node type.
			// So that we can use ajson library to resolve value path.
			root, err := ajson.Unmarshal(b)
			if err != nil {
				return err
			}
			if vals == nil {
				vals = []*ajson.Node{root}
			} else {
				vals = append(vals, root)
			}

		}
		k.metricsMap[key].Data = vals
	}
	return nil
}

func (k *kMetrics) Values(metric string) (Result, error) {

	res := Result{Vals: []interface{}{}, LabelValues: [][]string{}}

	// Iterate over the Data associated with the metric value.
	for _, val := range k.metricsMap[metric].Data {
		// Resolve the Value.
		v, err := ajson.Eval(val, k.metricsMap[metric].Path)
		if err != nil {
			log.Info(err.Error())
			continue
		}
		result, err := v.Value()
		if err != nil {
			log.Info(err.Error())
			continue
		}
		res.Vals = append(res.Vals, result)

		lValues := []string{}
		// Iterate over the label paths of the metric to resolve
		for _, lPath := range k.metricsMap[metric].LabelPath {
			// This helps us in setting constant labels.
			if lPath[0] != '$' {
				lValues = append(lValues, lPath)
				continue
			}
			// Resolve the Value.
			v, err := ajson.Eval(val, lPath)
			if err != nil {
				return res, err
			}
			result, err := v.Value()
			if err != nil {
				return res, err
			}

			str, ok := result.(string)
			if !ok {
				lValues = append(lValues, "")
			} else {
				lValues = append(lValues, str)
			}

		}
		res.LabelValues = append(res.LabelValues, lValues)
	}

	return res, nil
}
