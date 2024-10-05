package aichaosgenerator

import (
	"context"
	"errors"
	"fmt"

	"github.com/k8sgpt-ai/k8sgpt/pkg/ai"
	"github.com/k8sgpt-ai/k8sgpt/pkg/cache"
	"github.com/k8sgpt-ai/k8sgpt/pkg/common"
	"github.com/k8sgpt-ai/k8sgpt/pkg/kubernetes"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
)

type ChaosGenerator struct {
	Context            context.Context
	Client             *kubernetes.Client
	AIClient           ai.IAI
	Results            []common.Result
	Errors             []string
	Cache              cache.ICache
	MaxConcurrency     int
	AnalysisAIProvider string // The name of the AI Provider used for this generation
}

func NewAIChaosGenerator(
	backend string,
	filePath string,
	fileName string,
) (*ChaosGenerator, error) {
	// Get kubernetes client from viper.
	kubecontext := viper.GetString("kubecontext")
	kubeconfig := viper.GetString("kubeconfig")
	client, err := kubernetes.NewClient(kubecontext, kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("initialising kubernetes client: %w", err)
	}

	// Load remote cache if it is configured.
	cache, err := cache.GetCacheConfiguration()
	if err != nil {
		return nil, err
	}

	/*if noCache {
		cache.DisableCache()
	}*/

	cg := &ChaosGenerator{
		Context: context.Background(),
		Client:  client,
		Cache:   cache,
	}
	/*if !explain {
		// Return early if AI use was not requested.
		return a, nil
	}*/

	var configAI ai.AIConfiguration
	if err := viper.UnmarshalKey("ai", &configAI); err != nil {
		return nil, err
	}

	if len(configAI.Providers) == 0 {
		return nil, errors.New("AI provider not specified in configuration. Please run k8sgpt auth")
	}
	// Backend string will have high priority than a default provider
	// Hence, use the default provider only if the backend is not specified by the user.
	if configAI.DefaultProvider != "" && backend == "" {
		backend = configAI.DefaultProvider
	}

	if backend == "" {
		backend = "openai"
	}

	var aiProvider ai.AIProvider
	for _, provider := range configAI.Providers {
		if backend == provider.Name {
			aiProvider = provider
			break
		}
	}

	if aiProvider.Name == "" {
		return nil, fmt.Errorf("AI provider %s not specified in configuration. Please run k8sgpt auth", backend)
	}

	aiClient := ai.NewClient(aiProvider.Name)
	//customHeaders := util.NewHeaders(httpHeaders)
	//aiProvider.CustomHeaders = customHeaders
	if err := aiClient.Configure(&aiProvider); err != nil {
		return nil, err
	}
	cg.AIClient = aiClient
	cg.AnalysisAIProvider = aiProvider.Name

	corev1.AddToScheme(cg.Client.CtrlClient.Scheme())
	var serviceList corev1.ServiceList
	if err = cg.Client.CtrlClient.List(cg.Context, &serviceList); err != nil {
		return nil, fmt.Errorf("failed to get services from the cluster: %w", err)
	}
	svcContext := ai.ServiceAPIContext{Services: serviceList.Items}
	cg.Context = context.WithValue(cg.Context, "svcContext", svcContext)

	return cg, nil
}

func (cg *ChaosGenerator) Close() {
	if cg.AIClient == nil {
		return
	}
	cg.AIClient.Close()
}

func (cg *ChaosGenerator) GenerateChaos(prompt string) (string, error) {

	response, err := cg.AIClient.GetCompletion(cg.Context, prompt)
	if err != nil {
		return "", err
	}
	// fmt.Printf("SURYA: %v", response)
	return response, nil
}
