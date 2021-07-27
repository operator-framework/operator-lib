package jobharvest

import (
	"k8s.io/client-go/kubernetes"
)

// NewControllerClientGo returns a controller with a Harvester registered for each opt in opts.
func NewControllerClientGo(k8sClient kubernetes.Interface, opts ...*HarvesterOptions) (HarvestController, error) {
	hc := &harvestController{
		k8sClient: k8sClient,
		hrvs:      make(harvesters),
	}

	for _, opt := range opts {
		if _, err := hc.Create(opt); err != nil {
			return nil, err
		}
	}

	return hc, nil
}
