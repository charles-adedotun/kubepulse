package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/kubepulse/kubepulse/pkg/health"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check [check-name]",
	Short: "Run a specific health check",
	Long: `Run a specific health check and display the results.
Available checks: pod-health, node-health`,
	Args: cobra.ExactArgs(1),
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
	checkCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to check (for pod checks)")
}

func runCheck(cmd *cobra.Command, args []string) error {
	client := GetK8sClient()
	if client == nil {
		return fmt.Errorf("kubernetes client not initialized")
	}

	checkName := args[0]
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var result interface{}
	var err error

	switch checkName {
	case "pod-health":
		check := health.NewPodHealthCheck()
		if namespace != "" {
			if err := check.Configure(map[string]interface{}{
				"namespace": namespace,
			}); err != nil {
				return fmt.Errorf("failed to configure pod check: %w", err)
			}
		}
		result, err = check.Check(ctx, client)

	case "node-health":
		check := health.NewNodeHealthCheck()
		result, err = check.Check(ctx, client)

	default:
		return fmt.Errorf("unknown check: %s", checkName)
	}

	if err != nil {
		return fmt.Errorf("check failed: %w", err)
	}

	// Display result
	fmt.Printf("Check: %s\n", checkName)
	fmt.Printf("Result: %+v\n", result)

	return nil
}
