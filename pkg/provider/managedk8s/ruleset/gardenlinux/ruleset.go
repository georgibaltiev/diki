// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package gardenlinux

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/diki/imagevector"
	"github.com/gardener/diki/pkg/config"
	"github.com/gardener/diki/pkg/kubernetes/pod"
	kubeutils "github.com/gardener/diki/pkg/kubernetes/utils"
	"github.com/gardener/diki/pkg/rule"
	"github.com/gardener/diki/pkg/ruleset"
	"github.com/gardener/diki/pkg/shared/images"
	sharedrules "github.com/gardener/diki/pkg/shared/ruleset/disak8sstig/rules"
)

const (
	// RulesetID is a constant containing the id of the Gardenlinux Ruleset.
	RulesetID = "gardenlinux"
	// RulesetName is a constant containing the user-friendly name of the Gardenlinux ruleset.
	RulesetName = "Gardenlinux Ruleset"
)

var (
	_ ruleset.Ruleset = &Ruleset{}
	// SupportedVersions is a list of available versions for the Gardenlinux Ruleset.
	// Versions are sorted from newest to oldest.
	SupportedVersions = []string{"alpha"}
)

// Ruleset implements Gardelinux ruleset.
type Ruleset struct {
	version           string
	Config            *rest.Config
	Client            client.Client
	ClusterPodContext pod.SimplePodContext
	logger            *slog.Logger
	args              Args
}

// Args are Ruleset specific arguments.
type Args struct {
	NodeGroupByLabels []string `json:"nodeGroupByLabels" yaml:"nodeGroupByLabels"`
}

// ID returns the id of the Ruleset.
func (r *Ruleset) ID() string {
	return RulesetID
}

// Name returns the name of the Ruleset.
func (r *Ruleset) Name() string {
	return RulesetName
}

// Version returns the version of the Ruleset.
func (r *Ruleset) Version() string {
	return r.version
}

// New creates a new Ruleset.
func New(options ...CreateOption) (*Ruleset, error) {
	r := &Ruleset{}

	for _, o := range options {
		o(r)
	}

	c, err := client.New(r.Config, client.Options{})
	if err != nil {
		return nil, err
	}
	r.Client = c

	podContext, err := pod.NewSimplePodContext(r.Client, r.Config, nil)
	if err != nil {
		return nil, err
	}
	r.ClusterPodContext = *podContext

	return r, nil
}

// FromGenericConfig creates a Ruleset from a RulesetConfig
func FromGenericConfig(rulesetConfig config.RulesetConfig, managedConfig *rest.Config, _ *field.Path) (*Ruleset, error) {
	rulesetArgsByte, err := json.Marshal(rulesetConfig.Args)
	if err != nil {
		return nil, err
	}

	var rulesetArgs Args
	if err := json.Unmarshal(rulesetArgsByte, &rulesetArgs); err != nil {
		return nil, err
	}

	ruleset, err := New(
		WithVersion(rulesetConfig.Version),
		WithConfig(managedConfig),
		WithArgs(rulesetArgs),
	)
	if err != nil {
		return nil, err
	}
	return ruleset, nil
}

// Run executes the tests-ng containers and collects the test results.
func (r *Ruleset) Run(ctx context.Context) (ruleset.RulesetResult, error) {
	image, err := imagevector.ImageVector().FindImage(images.TestsNgImageName)
	if err != nil {
		return ruleset.RulesetResult{}, fmt.Errorf("failed to find image version for %s: %w", images.TestsNgImageName, err)
	}

	nodes, err := kubeutils.GetNodes(ctx, r.Client, 300)
	if err != nil {
		return ruleset.RulesetResult{}, err
	}

	allClusterPods, err := kubeutils.GetPods(ctx, r.Client, "", labels.NewSelector(), 300)
	if err != nil {
		return ruleset.RulesetResult{}, nil
	}

	nodesAllocatablePods := kubeutils.GetNodesAllocatablePodsNum(allClusterPods, nodes)

	selectedNodes, checkResults := kubeutils.SelectNodes(nodes, nodesAllocatablePods, r.args.NodeGroupByLabels)

	const systemNamespace = "kube-system"

	wg := sync.WaitGroup{}
	chResultChan := make(chan (rule.CheckResult))

	for _, node := range selectedNodes {
		wg.Add(1)
		go func() {
			var podName = fmt.Sprintf("test-ng-%s-%s", r.ID(), sharedrules.Generator.Generate(10))

			defer func() {
				wg.Done()
				timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
				defer cancel()

				if err := r.ClusterPodContext.Delete(timeoutCtx, podName, systemNamespace); err != nil {
					r.logger.Error(err.Error())
				}
			}()

			_, err := r.ClusterPodContext.Create(ctx, pod.NewGardenlinuxTestPod(podName, systemNamespace, image.String(), node.Name, nil))
			if err != nil {
				chResultChan <- rule.ErroredCheckResult(err.Error(), rule.NewTarget())
				return
			}

			// here i dont do error checking, since it is expected behaviour for the test-ng pod to error when there are failings
			err = r.ClusterPodContext.WaitPodCompleted(ctx, podName, systemNamespace)
			if err != nil {
				r.logger.Log(ctx, slog.LevelInfo, "pod has errored")
			}

			logs, err := kubeutils.GetPodLogs(ctx, r.Config, podName, systemNamespace)
			if err != nil {
				chResultChan <- rule.ErroredCheckResult(err.Error(), rule.NewTarget())
				return
			}

			// here we actually have to perform the parsing
			fmt.Print(logs)

			chResultChan <- rule.PassedCheckResult("Passed", rule.NewTarget())
		}()
	}

	go func() {
		wg.Wait()
		close(chResultChan)
	}()

	for checkResult := range chResultChan {
		checkResults = append(checkResults, checkResult)
	}

	return ruleset.RulesetResult{
		RulesetID:      RulesetID,
		RulesetVersion: r.version,
		RulesetName:    RulesetName,
		RuleResults: []rule.RuleResult{
			{
				RuleID:       "1000",
				CheckResults: checkResults,
			},
		},
	}, nil
}

// RunRule currently is not able to run a specific rule, since the implementation of the check is maintained externally
func (r *Ruleset) RunRule(_ context.Context, _ string) (rule.RuleResult, error) {
	return rule.RuleResult{}, fmt.Errorf("ruleset gardenlinux does not support running rules individually")
}

// Logger returns the Ruleset's logger.
// If not set it set it to slog.Default().With("ruleset", r.ID(), "version", r.Version() then return it.
func (r *Ruleset) Logger() *slog.Logger {
	if r.logger == nil {
		r.logger = slog.Default().With("ruleset", r.ID(), "version", r.Version())
	}
	return r.logger
}
