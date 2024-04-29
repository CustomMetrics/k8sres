package resconfigs

import (
	_ "embed"
)

//go:embed res.yaml
var K8sResourceMetricYaml string
