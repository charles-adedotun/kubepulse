package main

import (
	"fmt"
	"os"

	"github.com/kubepulse/kubepulse/cmd/kubepulse/commands"
	"k8s.io/klog/v2"
)

func main() {
	// Initialize klog
	klog.InitFlags(nil)

	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
