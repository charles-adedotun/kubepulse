package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

var (
	cfgFile       string
	kubeconfig    string
	contextName   string
	k8sClient     kubernetes.Interface
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "kubepulse",
	Short: "Intelligent Kubernetes health monitoring with ML-powered anomaly detection",
	Long: `KubePulse is a lightweight, intelligent Kubernetes health monitoring tool
that combines traditional threshold-based monitoring with ML-powered anomaly detection.

It provides instant "traffic light" health status for your clusters while
eliminating alert fatigue through smart, context-aware monitoring.`,
	Version: "0.1.0",
}

// Execute adds all child commands to the root command and sets flags appropriately
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kubepulse.yaml)")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file")
	rootCmd.PersistentFlags().StringVar(&contextName, "context", "", "kubernetes context to use")

	// Bind flags to viper
	if err := viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flag: %v\n", err)
	}
	if err := viper.BindPFlag("context", rootCmd.PersistentFlags().Lookup("context")); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flag: %v\n", err)
	}
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Search config in home directory with name ".kubepulse" (without extension)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kubepulse")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Initialize Kubernetes client
	initK8sClient()
}

// initK8sClient initializes the Kubernetes client
func initK8sClient() {
	var kubeconfigPath string

	// Try viper config first
	if path := viper.GetString("kubeconfig"); path != "" {
		kubeconfigPath = path
	} else if kubeconfig != "" {
		kubeconfigPath = kubeconfig
	} else {
		// Use default kubeconfig path
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	// Build config with context override if specified
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfigPath
	
	configOverrides := &clientcmd.ConfigOverrides{}
	
	// Use specified context if provided
	selectedContext := viper.GetString("context")
	if selectedContext != "" {
		configOverrides.CurrentContext = selectedContext
	}
	
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		configOverrides,
	)
	
	config, err := clientConfig.ClientConfig()
	if err != nil {
		// Try in-cluster config
		config, err = clientcmd.BuildConfigFromFlags("", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error building kubeconfig: %v\n", err)
			return
		}
	}

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
		return
	}

	k8sClient = clientset
}

// GetK8sClient returns the initialized Kubernetes client
func GetK8sClient() kubernetes.Interface {
	return k8sClient
}
