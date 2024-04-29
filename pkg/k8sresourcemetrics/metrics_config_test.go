package k8sresmetric

import (
	"testing"

	supCfg "github.com/CustomMetrics/k8sres/resconfig"
	"github.com/spyzhov/ajson"
	"github.com/stretchr/testify/assert"
)

var jsonStr1 string = `
{
	"value": 3,
	"label" : "test"
}
`
var jsonStr2 string = `
{
	"value": 4,
	"label": "test1"
}
`

func TestKMetricsValue(t *testing.T) {
	var rNodes []*ajson.Node
	rNode1, err := ajson.Unmarshal([]byte(jsonStr1))
	assert.Nil(t, err)
	rNode2, err := ajson.Unmarshal([]byte(jsonStr2))
	assert.Nil(t, err)

	rNodes = append(rNodes, rNode1, rNode2)

	mInf := &MetricsInfo{
		Data:      rNodes,
		Path:      "$.value",
		LabelPath: []string{"$.label"},
		LabelKeys: []string{"foo"},
	}
	nMap := make(map[string]*MetricsInfo)
	nMap["metric"] = mInf
	km := &kMetrics{
		metricsMap: nMap,
	}

	r, err := km.Values("metric")
	assert.Equal(t, 1, len(km.LabelNames("metric")))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(r.Vals))
}

func TestSet(t *testing.T) {
	err := SetCollectors(supCfg.K8sResourceMetricYaml)
	assert.Nil(t, err)

}
