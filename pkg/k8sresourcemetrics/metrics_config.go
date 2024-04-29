package k8sresmetric

import (
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(rscheme))
}

var resourceMap map[string]schema.GroupVersionKind
var shortNamesMap map[string]schema.GroupVersionKind
var rscheme = runtime.NewScheme()

type MetricsConfig struct {
	// All fields below must be exported (start with a capital letter)
	// so that the yaml.UnmarshalStrict() method can set them.
	Name       string `yaml:"name"`
	Help       string `yaml:"help"`
	MetricType string `yaml:"type"`
	Properties struct {
		PropertyType string            `yaml:"type"`
		Object       string            `yaml:"object"`
		Value        string            `yaml:"value"`
		Unit         string            `yaml:"unit"`
		Labels       map[string]string `yaml:"labels"`
	} `yaml:"properties"`
}

type ExporterConfig struct {
	Metrics []MetricsConfig `yaml:"metrics"`
}

func (e *ExporterConfig) Objects() []string {
	var objects []string
	for _, m := range e.Metrics {
		found := false
		for _, obj := range objects {
			if obj == m.Properties.PropertyType {
				found = true
				break
			}
		}
		if !found {
			objects = append(objects, m.Properties.PropertyType)
		}
	}

	return objects
}

type ResourceInfo struct {
	Type   string
	Data   interface{}
	Path   map[string][]interface{}
	Labels map[string]string
}

// Map Resource name to ResourceInfo.
type ResourceInfoMap map[string]*ResourceInfo

type ResourceMetricsCollector struct {
	ExpConfig   *ExporterConfig
	ResourceMap ResourceInfoMap
	sync.Mutex
}

var (
	dClient *discovery.DiscoveryClient
	cl      client.Client
)

// This function sets the map containing resource name and its corresponding
// gvk.
func setGVKMap() error {
	// Initialize the Map
	resourceMap = make(map[string]schema.GroupVersionKind)
	shortNamesMap = make(map[string]schema.GroupVersionKind)

	// List resources on the server
	_, resourceList, err := discovery.ServerGroupsAndResources(dClient)
	if err != nil {
		log.Errorf("Failed to list resources on server: %v", err)
		return err
	}
	var version string
	// Iterate over resource list to set the Map
	for _, resource := range resourceList {

		gv, _ := schema.ParseGroupVersion(resource.GroupVersion)
		// The apiversion seems to be like v1 or astra.netapp.io/v1aplha1
		// We just need v1 or v1aplha1
		vals := strings.Split(resource.GroupVersion, "/")
		if len(vals) > 1 {
			version = vals[1]
		} else {
			version = vals[0]
		}
		// Iterate over resource
		for _, apiRes := range resource.APIResources {
			// See if the resource kind is present
			_, ok := resourceMap[apiRes.Kind]
			if !ok {
				// Set map.
				resourceMap[apiRes.Kind] = schema.GroupVersionKind{
					Group:   gv.Group,
					Kind:    apiRes.Kind,
					Version: version,
				}
			}
			if len(apiRes.ShortNames) > 0 {
				shortNamesMap[apiRes.ShortNames[0]] = resourceMap[apiRes.Kind]
			}

		}
	}
	return nil
}

func getGVK(resource string) (schema.GroupVersionKind, error) {

	v, ok := resourceMap[resource]
	if !ok {
		v, ok = shortNamesMap[resource]
		if !ok {
			return schema.GroupVersionKind{}, fmt.Errorf("GVK not found for resource %s", resource)
		}
		return v, nil
	}
	return v, nil
}

// Set neccesary k8s client.
func SetClients(rCfg *rest.Config) (err error) {

	dClient, err = discovery.NewDiscoveryClientForConfig(rCfg)
	if err != nil {
		log.Errorf("Error building discovery client: %v", err.Error())
		return err
	}
	cl, err = client.New(rCfg, client.Options{Scheme: rscheme})
	if err != nil {
		log.Error("Failed to create k8s client: ", err)
		return err
	}

	return setGVKMap()
}
