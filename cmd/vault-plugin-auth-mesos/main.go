package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault/helper/pluginutil"
	"github.com/hashicorp/vault/logical/plugin"

	backend "github.com/praekeltfoundation/vault-plugin-auth-mesos"
	"github.com/praekeltfoundation/vault-plugin-auth-mesos/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "Version")
	flag.Parse()

	if *versionFlag {
		reportVersion()
		return
	}

	runPlugin()
}

func reportVersion() {
	fmt.Println("Git Commit:", version.GitCommit)
	fmt.Println("Version:", version.Version)
	if version.VersionPrerelease != "" {
		fmt.Println("Version PreRelease:", version.VersionPrerelease)
	}
}

func runPlugin() {
	apiClientMeta := &pluginutil.APIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:])

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := pluginutil.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.Serve(&plugin.ServeOpts{
		BackendFactoryFunc: backend.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	}); err != nil {
		log.Fatal(err)
	}
}
