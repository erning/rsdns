package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/erning/rsdns/internal/rsdns"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "rsdns",
	Short: "Really Simple DNS",
	Run: func(cmd *cobra.Command, args []string) {
		rsdns.Serve()
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&configFile, "config", "", "config file")

	flags.String("dns.zone", "dynamic.wacao.com.", "zone")
	flags.String("dns.addr", ":8053", "listen address of dns server")

	flags.String("http.base", "/", "base path of http api")
	flags.String("http.addr", ":8080", "listen address of http api")

	flags.String("data", "rsdns-data.json", "data file")

	flags.BoolP("verbose", "v", false, "show debug messages")
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("rsdns")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/")
	}

	_ = viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))
	_ = viper.BindPFlag("dns.zone", rootCmd.Flags().Lookup("dns.zone"))
	_ = viper.BindPFlag("dns.addr", rootCmd.Flags().Lookup("dns.addr"))
	_ = viper.BindPFlag("http.base", rootCmd.Flags().Lookup("http.base"))
	_ = viper.BindPFlag("http.addr", rootCmd.Flags().Lookup("http.addr"))

	_ = viper.BindPFlag("data", rootCmd.Flags().Lookup("data"))

	// viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
		} else {
			// Config file was found but another error was produced
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		}
	}

	zone := viper.GetString("dns.zone")
	zone = strings.Trim(zone, ".")
	viper.Set("dns.zone", zone+".")

	base := viper.GetString("http.base")
	base = strings.TrimSuffix(base, "/")
	viper.Set("http.base", base)

	// fmt.Printf("%#v\n", viper.AllSettings())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
