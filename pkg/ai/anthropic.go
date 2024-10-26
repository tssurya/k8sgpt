package ai

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/liushuangls/go-anthropic/v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/utils/ptr"
)

const anthropicClientName = "claude"

var attachYamlContext bool

type NetworkPolicyAPIContext struct {
	Namespaces    []corev1.Namespace
	NetworkPolicy []networkingv1.NetworkPolicy
	StatefulSets  []appsv1.StatefulSet
}

type ClaudeClient struct {
	nopCloser

	client      *anthropic.Client
	model       string
	temperature float32
	topP        float32
	topK        int32
	maxTokens   int
}

func (c *ClaudeClient) Configure(config IAIConfig) error {
	token := config.GetPassword()

	client := anthropic.NewClient(token)
	if client == nil {
		return errors.New("error creating OpenAI client")
	}
	c.client = client
	c.model = config.GetModel()
	c.temperature = config.GetTemperature()
	c.topP = config.GetTopP()
	c.maxTokens = 2048
	return nil
}

func (c *ClaudeClient) GetCompletion(ctx context.Context, prompt string) (string, error) {

	req := anthropic.MessagesRequest{
		Model:       anthropic.ModelClaude3Dot5Sonnet20241022,
		System:      "You are a Kubernetes networking and NetworkPolicy API expert. You have vast experience with NetworkPolicy chaos tests.",
		Temperature: ptr.To(c.temperature),
		TopP:        ptr.To(c.topP),
		TopK:        ptr.To[int](int(c.topK)),
		MaxTokens:   maxToken,
	}

	req.Messages = []anthropic.Message{
		anthropic.NewUserTextMessage(prompt),
	}

	resp, err := c.client.CreateMessages(ctx, req)
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		// continue
		return "", err
	}

	fmt.Printf("%s\n\n", resp.Content[0].GetText())
	req.Messages = append(req.Messages, anthropic.Message{Role: resp.Role, Content: resp.Content})
	fmt.Print("> ")

	// fmt.Println("Conversation")
	// fmt.Println("---------------------")
	// fmt.Print("> ")
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		attachYamlContext = false
		input := s.Text()
		if input == "exit" {
			return "", nil
		}
		if strings.HasPrefix(input, "look at my cluster") {
			input = strings.TrimPrefix(input, "look at my cluster")
			attachYamlContext = true
		}
		if attachYamlContext {
			a := ctx.Value("npContext")
			b := a.(NetworkPolicyAPIContext)
			input += fmt.Sprintf("applied namespaces in my cluster:\n%v; applied stateful-sets in my cluster:\n%v; applied network-policies in my cluster:\n%v", b.Namespaces, b.StatefulSets, b.NetworkPolicy)
		}
		req.Messages = append(req.Messages, anthropic.NewUserTextMessage(input))
		resp, err := c.client.CreateMessages(ctx, req)
		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)
			// continue
			return "", err
		}
		fmt.Printf("%s\n\n", resp.Content[0].GetText())
		req.Messages = append(req.Messages, anthropic.Message{Role: resp.Role, Content: resp.Content})
		fmt.Print("> ")
	}
	return "", nil

	// fmt.Printf("Usage: %+v\n", resp.Usage)
	// resp.Content[0].
	// return resp.Content[0].GetText(), nil
}

func (c *ClaudeClient) GetName() string {
	return anthropicClientName
}
