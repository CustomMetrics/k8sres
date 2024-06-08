package k8sresmetric

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spyzhov/ajson"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MetricsInfo struct {
	Data          []*ajson.Node
	Path          string
	Obj           string
	FieldSelector fields.Selector
	LabelSelector labels.Selector
	Namespace     string
	Iterator      string
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
	var err error
	k.metricsMap[m.Name] = &MetricsInfo{
		Obj:       m.Properties.Object,
		Path:      m.Properties.Value,
		Namespace: m.Properties.Namespace,
		LabelKeys: []string{},
		Data:      []*ajson.Node{},
	}

	isKeyPresent := false
	for key, path := range m.Properties.Labels {
		if strings.HasPrefix(path, "key") {
			isKeyPresent = true
		}
		k.metricsMap[m.Name].LabelKeys = append(k.metricsMap[m.Name].LabelKeys, key)
		k.metricsMap[m.Name].LabelPath = append(k.metricsMap[m.Name].LabelPath, path)
	}

	if m.Properties.FieldSelector != "" {
		k.metricsMap[m.Name].FieldSelector, err = fields.ParseSelector(m.Properties.FieldSelector)
		if err != nil {
			return err
		}
	}

	if m.Properties.LabelSelector != "" {
		k.metricsMap[m.Name].LabelSelector, err = labels.Parse(m.Properties.LabelSelector)
		if err != nil {
			return err
		}
	}

	if isKeyPresent {
		// Move the "key" label to the end.
		// It becomes easier to manipulate label values.
		for i, path := range k.metricsMap[m.Name].LabelPath {
			temp := k.metricsMap[m.Name].LabelPath
			temp2 := k.metricsMap[m.Name].LabelKeys

			if strings.HasPrefix(path, "key") {
				tempKey := temp2[i]
				temp = append(temp[:i], temp[i+1:]...)
				temp = append(temp, path)

				temp2 = append(temp2[:i], temp2[i+1:]...)
				temp2 = append(temp2, tempKey)
			}
			k.metricsMap[m.Name].LabelPath = temp
			k.metricsMap[m.Name].LabelKeys = temp2
		}
	}

	return nil
}

func (k *kMetrics) LabelNames(metric string) []string {
	return k.metricsMap[metric].LabelKeys
}

func (k *kMetrics) Update() error {

	// Clear existing cr list.
	k.crMap = make(map[string]unstructured.UnstructuredList)
	for key, val := range k.metricsMap {
		v, err := getGVK(val.Obj)
		if err != nil {
			log.Error(err)
			continue
		}

		// Check if the object related to this GVK has been scraped before.
		if _, ok := k.crMap[v.Kind]; !ok {
			u := unstructured.UnstructuredList{}
			u.SetGroupVersionKind(v)

			err = cl.List(context.Background(), &u, &client.ListOptions{Namespace: "", FieldSelector: val.FieldSelector, LabelSelector: val.LabelSelector})
			if err != nil {
				log.Error(err.Error())
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

		//  Handle "val" and "key" keyword.
		if strings.HasPrefix(k.metricsMap[metric].Path, "val") {

			temp := k.metricsMap[metric].Path
			path := temp[4 : len(temp)-1]

			v, err := ajson.Eval(val, path)
			if err != nil {
				log.Info(err.Error())
				continue
			}

			vMap, err := v.Unpack()
			if err != nil {
				log.Info(err.Error())
				continue
			}

			// Try to coerce it to map[string]interface{} (interface because value can be anything.)
			if _, ok := vMap.(map[string]interface{}); ok {

				for _, vVal := range vMap.(map[string]interface{}) {
					res.Vals = append(res.Vals, vVal)
				}
			}

		} else {
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
		}

		lValues := []string{}

		// Flag to check if we have key or val
		// in the metric
		flag := false
		// Iterate over the label paths of the metric to resolve
		for _, lPath := range k.metricsMap[metric].LabelPath {

			// Handle "key" keyword.
			if strings.HasPrefix(lPath, "key") {
				// i = index
				path := lPath[4 : len(lPath)-1]

				v, err := ajson.Eval(val, path)
				if err != nil {
					log.Info(err.Error())
					continue
				}

				vMap, err := v.Unpack()
				if err != nil {
					log.Info(err.Error())
					continue
				}

				// Try to coerce it to map[string]interface{} (interface because value can be anything.)
				if _, ok := vMap.(map[string]interface{}); ok {
					temp := []string{}
					for key, _ := range vMap.(map[string]interface{}) {
						flag = true
						temp = append(lValues, key)
						res.LabelValues = append(res.LabelValues, temp)
					}
				}
				continue
			}

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
		// If labels have been modified by the internal loop
		// avoid appending lableValues as labels has been taken care of.
		if !flag {
			res.LabelValues = append(res.LabelValues, lValues)
		}

	}

	return res, nil
}
