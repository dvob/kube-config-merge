package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(0)
	}
}

func run() error {
	var (
		loadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
		override     = false
		server       string
		overrideName string
		prefix       string
	)

	pflag.StringVar(&loadingRules.ExplicitPath, "kubeconfig", loadingRules.ExplicitPath, "Path to the kubeconfig file, where other config should be merged into. if not specified default locations will be used.")
	pflag.BoolVar(&override, "override", override, "Overwrite existing clusters, contexts and users in target config.")
	pflag.StringVar(&server, "server", server, "Overwrite the server url from the source with this particular server. This usally only makes sense if you have a single cluster in the source config.")
	pflag.StringVar(&overrideName, "name", overrideName, "Put cluster, context and user in the target config under that specific name. Only works if you don't have more than one of each in source config.")
	pflag.StringVar(&prefix, "prefix", prefix, "Prefix names in target config with prefix.")
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [SOURCE]:\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, `Merges Kubernetes configuration from file SOURCE into the current
configuration. It uses the default kubectl config locations or you can
explicitly specify a target config using --kubeconfig flag. If no SOURCE is
specified it is read from standard input.

Examples:
kube-config-merge some-kubeconfig.yaml

kube-config-merge --kubeconfig my-target-config.yaml some-kubeconfig.yaml

ssh k3s-host sudo cat /etc/rancher/k3s/k3s.yaml | kube-config-merge --name k3s --server https://k3s-host

Options:`)
		pflag.PrintDefaults()
	}

	pflag.Parse()

	//
	// read source
	//
	sourceConfigPath := pflag.Arg(0)

	var sourceConfigReader io.Reader
	if sourceConfigPath == "" || sourceConfigPath == "-" {
		sourceConfigReader = os.Stdin
	} else {
		var err error
		sourceConfigReader, err = os.Open(sourceConfigPath)
		if err != nil {
			return fmt.Errorf("failed to open source config: %w", err)
		}
	}

	sourceConfigData, err := io.ReadAll(sourceConfigReader)
	if err != nil {
		return fmt.Errorf("failed to read source config: %w", err)
	}

	sourceConfig, err := clientcmd.Load(sourceConfigData)
	if err != nil {
		return fmt.Errorf("failed to parse source config: %w", err)
	}

	//
	// read target
	//
	config, err := loadingRules.GetStartingConfig()
	if err != nil {
		return fmt.Errorf("failed to open target config: %w", err)
	}

	//
	// merge source into target
	//
	if overrideName != "" && (len(sourceConfig.Clusters) > 1 || len(sourceConfig.Contexts) > 1 || len(sourceConfig.AuthInfos) > 1) {
		return fmt.Errorf("you can't use --name to override name for source configurations with more than one cluster, context or user")
	}

	for name, cluster := range sourceConfig.Clusters {
		if config.Clusters == nil {
			config.Clusters = make(map[string]*api.Cluster)
		}

		targetName := getTargetName(name, overrideName, prefix)
		if _, exists := config.Clusters[targetName]; exists && !override {
			return fmt.Errorf("cluster '%s' does already exist in target config", targetName)
		}

		newCluster := cluster.DeepCopy()

		if server != "" {
			newCluster.Server = server
		}

		config.Clusters[targetName] = newCluster
	}

	for name, context := range sourceConfig.Contexts {
		if config.Contexts == nil {
			config.Contexts = make(map[string]*api.Context)
		}

		targetName := getTargetName(name, overrideName, prefix)
		if _, exists := config.Contexts[targetName]; exists && !override {
			return fmt.Errorf("context '%s' does already exist in target config", targetName)
		}

		newContext := context.DeepCopy()
		newContext.Cluster = getTargetName(context.Cluster, overrideName, prefix)
		newContext.AuthInfo = getTargetName(context.AuthInfo, overrideName, prefix)

		config.Contexts[targetName] = newContext
	}

	for name, user := range sourceConfig.AuthInfos {
		if config.AuthInfos == nil {
			config.AuthInfos = make(map[string]*api.AuthInfo)
		}

		targetName := getTargetName(name, overrideName, prefix)
		if _, exists := config.AuthInfos[targetName]; exists && !override {
			return fmt.Errorf("user '%s' does already exist in target config", targetName)
		}

		newAuthInfo := user.DeepCopy()

		config.AuthInfos[targetName] = newAuthInfo
	}

	//
	// write target
	//
	err = clientcmd.ModifyConfig(loadingRules, *config, false)
	if err != nil {
		return fmt.Errorf("failed to modify configuration: %w", err)
	}

	return nil
}

func getTargetName(name string, overrideName string, prefix string) string {
	if overrideName != "" {
		return prefix + overrideName
	}
	return prefix + name
}
