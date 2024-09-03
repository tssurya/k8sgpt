package aichaos

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	aichaosgenerator "github.com/k8sgpt-ai/k8sgpt/pkg/aichaos_generator"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	backend       string
	chaosScenario string
	filePath      string
	fileName      string
)

// AIChaosCmd represents the problems command
var AIChaosCmd = &cobra.Command{
	Use:     "aichaos",
	Aliases: []string{"aichaos"},
	Short:   "This command will introduce chaos in your Kubernetes cluster",
	Long:    `This command will introduce problems within your Kubernetes cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create analysis configuration first.
		chaosGenerator, err := aichaosgenerator.NewAIChaosGenerator(
			backend,
			filePath,
			fileName,
		)

		if err != nil {
			color.Red("Error: %v", err)
			os.Exit(1)
		}
		defer chaosGenerator.Close()
		// Process template.
		var prompt string
		if fileName != "" {
			// Read the YAML file
			yamlContent, err := os.ReadFile(filePath + fileName)
			if err != nil {
				log.Fatalf("Failed to read YAML file: %v", err)
			}

			// Split the content into multiple documents
			documents := strings.Split(string(yamlContent), "---\n")

			var formattedYAML strings.Builder
			for _, doc := range documents {
				if strings.TrimSpace(doc) == "" {
					continue
				}

				var data map[interface{}]interface{}
				err = yaml.Unmarshal([]byte(doc), &data)
				if err != nil {
					log.Printf("Failed to parse YAML document: %v", err)
					continue
				}

				formattedYAML.WriteString("---\n")
				formattedYAML.WriteString(formatYAML(data, 0))
				formattedYAML.WriteString("\n")
			}
			//fmt.Printf("SURYA %v", formattedYAML)
			prompt = fmt.Sprintf("Act as a Chaos generator in a Kubernetes cluster. Read the yamls I have provided: %s "+
				"and introduce random pod label changes such that network policies don't behave as expected and what should"+
				" be allowed is denied and what should be denied is allowed", formattedYAML.String())
		} else {
			prompt = chaosScenario
		}
		_, err = chaosGenerator.GenerateChaos(prompt)
		if err != nil {
			color.Red("Error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	AIChaosCmd.Flags().StringVarP(&fileName, "filename", "f", "", "YAML File that contains the cluster config")
	// explain flag
	AIChaosCmd.Flags().StringVarP(&filePath, "filepath", "p", "", "File path for the yaml file")
	AIChaosCmd.Flags().StringVarP(&chaosScenario, "scenario", "s", "", "Explain Chaos Scenario")
	// add flag for backend
	AIChaosCmd.Flags().StringVarP(&backend, "backend", "b", "", "Backend AI provider")
}

func formatYAML(data interface{}, indent int) string {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		var builder strings.Builder
		for key, value := range v {
			builder.WriteString(strings.Repeat("  ", indent))
			builder.WriteString(fmt.Sprintf("%v:\n", key))
			builder.WriteString(formatYAML(value, indent+1))
		}
		return builder.String()
	case []interface{}:
		var builder strings.Builder
		for _, item := range v {
			builder.WriteString(strings.Repeat("  ", indent))
			builder.WriteString("- ")
			builder.WriteString(formatYAML(item, indent+1))
		}
		return builder.String()
	default:
		return fmt.Sprintf("%s%v\n", strings.Repeat("  ", indent), v)
	}
}
