package jobharvest

import (
	"context"
	"fmt"
	"io"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// HarvesterRegistrationKey is an annotation that should be set on a job if that
	// job's name may be random.
	HarvesterRegistrationKey = "job-harvester.operatorframework.io/registration-key"

	controllerName = "job-harvester"
)

// HarvesterOptions configures a new Harvester.
type HarvesterOptions struct {
	Name      string
	LogWriter LogWriter
	RunOnce   bool
	Cleanups  []func() error
}

// LogWriter writes logs from input stream io.Reader.
type LogWriter interface {
	WriteLogs(context.Context, io.Reader, corev1.Pod, string) error
}

// WriteLogsFunc implements LogWriter for functions.
type WriteLogsFunc func(context.Context, io.Reader, corev1.Pod, string) error

// WriteLogs calls f(args).
func (f WriteLogsFunc) WriteLogs(ctx context.Context, stream io.Reader, pod corev1.Pod, ctrName string) error {
	return f(ctx, stream, pod, ctrName)
}

// WriteLogsTo writes logs to w.
func WriteLogsTo(w io.Writer) LogWriter {
	return WriteLogsFunc(func(_ context.Context, stream io.Reader, _ corev1.Pod, _ string) error {
		_, err := io.Copy(w, stream)
		return err
	})
}

// HarvestController is a controller for harvesters.
type HarvestController interface {
	Create(*HarvesterOptions) (Harvester, error)
	Remove(*batchv1.Job) error
	RemoveNamed(string) error
}

// harvestController reconciles all harvesters.
type harvestController struct {
	k8sClient  kubernetes.Interface
	ctrlClient client.Client
	hrvs       harvesters
}

// Create creates a new Harvester under name. Name must either be the exact
// Job name or the string value for the HarvesterRegistrationKey annotation.
func (hc *harvestController) Create(opts *HarvesterOptions) (Harvester, error) {
	if hc.k8sClient == nil {
		return nil, fmt.Errorf("k8sClient must be set")
	}
	// if hc.ctrlClient == nil {
	// 	return nil, fmt.Errorf("ctrlClient must be set")
	// }

	h := &harvester{
		name:       opts.Name,
		k8sClient:  hc.k8sClient,
		ctrlClient: hc.ctrlClient,
	}

	if h.name = opts.Name; h.name == "" {
		return nil, fmt.Errorf("name must be set")
	}
	if h.lw = opts.LogWriter; h.lw == nil {
		return nil, fmt.Errorf("log reader must be configured")
	}

	if err := hc.hrvs.register(opts.Name, &controlledHarvester{
		harvester:  h,
		runOnce:    opts.RunOnce,
		cleanupFns: opts.Cleanups,
	}); err != nil {
		return nil, err
	}

	return h, nil
}

// Remove removes job from the HarvesterController.
func (hc *harvestController) Remove(job *batchv1.Job) error {
	h, has := hc.hrvs.get(job)
	if !has {
		return nil
	}

	cleanupErrs := []error{}
	for _, cleanup := range h.cleanupFns {
		if err := cleanup(); err != nil {
			cleanupErrs = append(cleanupErrs, err)
		}
	}
	if len(cleanupErrs) != 0 {
		return errors.NewAggregate(cleanupErrs)
	}

	delete(hc.hrvs, h.name)
	return nil
}

// RemoveNamed removes harvester keyed with name from the HarvesterController.
func (hc *harvestController) RemoveNamed(name string) error {
	job := batchv1.Job{}
	job.Name = name
	return hc.Remove(&job)
}

type controlledHarvester struct {
	*harvester
	runOnce    bool
	cleanupFns []func() error
}

type harvesters map[string]*controlledHarvester

func (hs harvesters) register(name string, h *controlledHarvester) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if _, registered := hs[name]; registered {
		return fmt.Errorf("harvester %q already registered", name)
	}
	hs[name] = h
	return nil
}

func (hs harvesters) get(job *batchv1.Job) (h *controlledHarvester, registered bool) {
	if h, registered = hs[job.Name]; !registered && job.Annotations != nil {
		if regValue := job.Annotations[HarvesterRegistrationKey]; regValue != "" {
			h, registered = hs[regValue]
		}
	}
	return
}
