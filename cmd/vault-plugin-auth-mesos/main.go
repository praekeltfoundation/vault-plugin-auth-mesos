package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/vault/helper/pluginutil"
	"github.com/hashicorp/vault/logical/plugin"

	"github.com/praekeltfoundation/vault-plugin-auth-mesos"
	"github.com/praekeltfoundation/vault-plugin-auth-mesos/version"
)

func main() {
	apiClientMeta := &pluginutil.APIClientMeta{}
	flags := apiClientMeta.FlagSet()
	versionFlag := flags.Bool("version", false, "Print version information and exit.")

	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatal(err)
	}

	if *versionFlag {
		fmt.Println(version.HumanReadable())
		return
	}

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := pluginutil.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.Serve(&plugin.ServeOpts{
		BackendFactoryFunc: mesosauth.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	}); err != nil {
		log.Fatal(err)
	}
}
