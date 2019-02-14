package main

import (
	"github.com/hfaulds/mirror/issues"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "mirror",
	Short: "Mirror repository",
}

func main() {
	viper.SetEnvPrefix("github")
	viper.AutomaticEnv()

	rootCmd.PersistentFlags().StringP("token", "t", "", "GitHub Token")
	viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))

	rootCmd.PersistentFlags().StringP("from", "f", "", "GitHub Repository to sync from")
	viper.BindPFlag("from", rootCmd.PersistentFlags().Lookup("from"))

	rootCmd.PersistentFlags().StringP("to-token", "", "", "PAT for target repository")
	viper.BindPFlag("to_token", rootCmd.PersistentFlags().Lookup("to-token"))

	rootCmd.PersistentFlags().StringP("to", "", "", "GitHub Repository to sync to")
	viper.BindPFlag("to", rootCmd.PersistentFlags().Lookup("to"))

	rootCmd.AddCommand(issues.Command())

	rootCmd.Execute()
}
