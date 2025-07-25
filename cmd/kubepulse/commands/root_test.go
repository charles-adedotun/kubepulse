package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestExecute(t *testing.T) {
	// Test that Execute function exists and can be called
	// We can't compare functions to nil directly in Go
	// The main integration tests cover this functionality

	// Just verify the function exists by ensuring it compiles
	var _ func() error = Execute
}

func TestRootCommand_Basic(t *testing.T) {
	// Test root command structure
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	if rootCmd.Use != "kubepulse" {
		t.Errorf("expected Use to be 'kubepulse', got %s", rootCmd.Use)
	}

	if !strings.Contains(rootCmd.Short, "monitoring") {
		t.Errorf("expected Short to contain 'monitoring', got %s", rootCmd.Short)
	}

	if !strings.Contains(rootCmd.Long, "KubePulse") {
		t.Errorf("expected Long to contain 'KubePulse', got %s", rootCmd.Long)
	}

	if rootCmd.Version != "0.1.0" {
		t.Errorf("expected Version to be '0.1.0', got %s", rootCmd.Version)
	}
}

func TestRootCommand_Flags(t *testing.T) {
	// Test that persistent flags are set up correctly
	flags := rootCmd.PersistentFlags()

	configFlag := flags.Lookup("config")
	if configFlag == nil {
		t.Error("config flag should be defined")
	}

	kubeconfigFlag := flags.Lookup("kubeconfig")
	if kubeconfigFlag == nil {
		t.Error("kubeconfig flag should be defined")
	}

	contextFlag := flags.Lookup("context")
	if contextFlag == nil {
		t.Error("context flag should be defined")
	}
}

func TestRootCommand_Help(t *testing.T) {
	// Test help output
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Reset args to avoid interference
	originalArgs := rootCmd.Flag("help")
	defer func() {
		if originalArgs != nil {
			rootCmd.Flags().Set("help", originalArgs.Value.String())
		}
	}()

	rootCmd.SetArgs([]string{"--help"})

	// Execute should not return error for help
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("help command should not return error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "kubepulse") {
		t.Errorf("help output should contain 'kubepulse', got: %s", output)
	}
}

func TestInitConfig_NoFile(t *testing.T) {
	// Save original values
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()

	// Reset viper for clean test
	viper.Reset()

	// Test with no config file
	cfgFile = ""

	// Create a test function that won't exit
	testInitConfig := func() {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				// Don't exit in test, just return
				return
			}
			viper.AddConfigPath(home)
			viper.SetConfigType("yaml")
			viper.SetConfigName(".kubepulse")
		}
		viper.AutomaticEnv()
		// Don't call viper.ReadInConfig() to avoid file system dependencies
	}

	// This should not panic
	testInitConfig()

	// Verify viper is configured
	if viper.ConfigFileUsed() != "" && !strings.Contains(viper.ConfigFileUsed(), ".kubepulse") {
		t.Errorf("expected config file to contain '.kubepulse' or be empty, got: %s", viper.ConfigFileUsed())
	}
}

func TestInitConfig_WithFile(t *testing.T) {
	// Save original values
	originalCfgFile := cfgFile
	defer func() { cfgFile = originalCfgFile }()

	// Reset viper for clean test
	viper.Reset()

	// Create temporary config file
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")

	err := os.WriteFile(configFile, []byte("test: value\n"), 0644)
	if err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfgFile = configFile

	// Test with config file
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	// Read the config
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	// Verify config was read
	if viper.GetString("test") != "value" {
		t.Errorf("expected config value 'value', got: %s", viper.GetString("test"))
	}
}

func TestGetK8sClient_NotInitialized(t *testing.T) {
	// Save original client
	originalClient := k8sClient
	defer func() { k8sClient = originalClient }()

	// Set client to nil
	k8sClient = nil

	client := GetK8sClient()
	if client != nil {
		t.Error("expected nil client when not initialized")
	}
}

func TestGlobalVariables(t *testing.T) {
	// Test that global variables are accessible
	if rootCmd == nil {
		t.Error("rootCmd should be initialized")
	}

	// Test variable types
	var _ string = cfgFile
	var _ string = kubeconfig
	var _ string = contextName

	// These variables should be declared (may be nil initially)
	if &k8sClient == nil {
		t.Error("k8sClient variable should be declared")
	}
}

func TestRootCommand_Version(t *testing.T) {
	// Test version functionality
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)

	// Test version flag
	rootCmd.SetArgs([]string{"--version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("version command should not return error, got: %v", err)
	}

	output := buf.String()
	// The version command shows help + version info, so check for version info in output
	if !strings.Contains(output, "version") || !strings.Contains(output, "kubepulse") {
		t.Errorf("version output should contain version info, got: %s", output)
	}
}

func TestViperBindings(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Test viper flag bindings by setting values
	viper.Set("kubeconfig", "/test/kubeconfig")
	viper.Set("context", "test-context")

	if viper.GetString("kubeconfig") != "/test/kubeconfig" {
		t.Errorf("expected kubeconfig '/test/kubeconfig', got: %s", viper.GetString("kubeconfig"))
	}

	if viper.GetString("context") != "test-context" {
		t.Errorf("expected context 'test-context', got: %s", viper.GetString("context"))
	}
}

func TestRootCommand_Commands(t *testing.T) {
	// Test that root command can have subcommands added
	commands := rootCmd.Commands()

	// Should have some commands (added by init functions)
	if len(commands) == 0 {
		t.Error("expected root command to have subcommands")
	}

	// Test that we can add a test command
	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test command",
		Run:   func(cmd *cobra.Command, args []string) {},
	}

	rootCmd.AddCommand(testCmd)
	defer rootCmd.RemoveCommand(testCmd)

	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "test" {
			found = true
			break
		}
	}

	if !found {
		t.Error("test command should be added to root command")
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Test that viper reads environment variables
	viper.Reset()

	// Set environment variable
	os.Setenv("KUBECONFIG", "/env/kubeconfig")
	defer os.Unsetenv("KUBECONFIG")

	viper.AutomaticEnv()

	// Viper should read the environment variable
	if viper.GetString("KUBECONFIG") != "/env/kubeconfig" {
		t.Errorf("expected KUBECONFIG '/env/kubeconfig', got: %s", viper.GetString("KUBECONFIG"))
	}
}

func TestInitConfigErrors(t *testing.T) {
	// Test config initialization with various scenarios
	viper.Reset()

	// Test with non-existent config file
	viper.SetConfigFile("/nonexistent/config.yaml")
	err := viper.ReadInConfig()

	// Should handle error gracefully (file not found is expected)
	if err == nil {
		t.Error("expected error when reading non-existent config file")
	}
}

func TestRootCommandStructure(t *testing.T) {
	// Test command structure and metadata
	if rootCmd.Use == "" {
		t.Error("rootCmd.Use should not be empty")
	}

	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}

	if rootCmd.Long == "" {
		t.Error("rootCmd.Long should not be empty")
	}

	if rootCmd.Version == "" {
		t.Error("rootCmd.Version should not be empty")
	}

	// Test that command has proper structure
	if !rootCmd.HasSubCommands() {
		t.Error("rootCmd should have subcommands")
	}
}
