package k8sresmetric

import (
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
}

type ResourceCollector interface {
	RegisterMetric(m MetricsConfig) error
	LabelNames(metrics string) []string
	Update() error
	Values(metric string) (Result, error)
}

// Result returns all the values related to a metric.
type Result struct {
	// Array of all the metric Values.
	Vals []interface{}
	// Array of the labels associated with the metric Value.
	LabelValues [][]string
}

type Collector struct {
	ResourceCollector
	MetricConfigList []MetricsConfig
	lastScrapeTime   time.Time
	sync.Mutex
}

var (
	// interval between each metric scrape in seconds.
	scrapeInterval float64 = 25
)

func NewResourceCollector(resType string) ResourceCollector {

	switch resType {
	case "kubernetes":
		return &kMetrics{metricsMap: make(map[string]*MetricsInfo)}
	}

	return nil
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {

	for _, m := range c.MetricConfigList {
		ch <- prometheus.NewDesc(m.Name, m.Help, c.LabelNames(m.Name), nil)
	}

}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {

	c.Lock()
	defer c.Unlock()

	// Refresh data only after scrape interval.
	t := time.Now()
	elapsed := t.Sub(c.lastScrapeTime)
	if elapsed.Seconds() >= scrapeInterval {
		c.Update()
		c.lastScrapeTime = time.Now()
	}

	for _, m := range c.MetricConfigList {
		// Get the value and labels associated with the metric.
		r, err := c.Values(m.Name)
		if err != nil {
			log.Errorf("error resolving metric %v", err)
			continue
		}
		// Range over the result.
		for key, val := range r.Vals {
			var v float64
			// convert the value based on the unit.
			v, err = ConvertUnit(m.Properties.Unit, val)
			if err != nil {
				v = 0.0
			}

			desc := prometheus.NewDesc(m.Name, m.Help, c.LabelNames(m.Name), nil)
			// Set Metric Type.
			metricType := prometheus.UntypedValue
			switch m.MetricType {
			case "counter":
				metricType = prometheus.CounterValue
			case "gauge":
				metricType = prometheus.GaugeValue
			}

			ch <- prometheus.MustNewConstMetric(desc,
				metricType,
				v,
				r.LabelValues[key]...)

		}
	}
}

func SetCollectors(config string) error {
	exp := &ExporterConfig{}

	err := yaml.Unmarshal([]byte(config), exp)
	if err != nil {
		return err
	}
	resMap := make(map[string]*Collector)
	// Iterate over
	for _, propType := range exp.Objects() {
		// Set Resource Collector
		resCol := NewResourceCollector(propType)
		// Create instance of the collector
		c := &Collector{ResourceCollector: resCol}

		resMap[propType] = c
	}

	// Iterate over all the metrics and register those
	// according to their property Type.
	for _, metric := range exp.Metrics {
		c, ok := resMap[metric.Properties.PropertyType]
		if !ok {
			continue
		}

		c.RegisterMetric(metric)
		c.MetricConfigList = append(c.MetricConfigList, metric)
	}

	for _, val := range resMap {
		prometheus.MustRegister(val)
	}

	return nil
}
