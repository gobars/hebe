/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package es

import (
	"github.com/spf13/cobra"
)

// esCmd represents the es command
var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Display nodes of cluster",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		cluster := cmd.Flag("cluster").Value.String()
		attrs, err := cmd.Flags().GetBool("attrs")
		if err != nil {
			panic(err)
		}
		if attrs {
			handleCatCommand(cluster, "nodeattrs")
			return
		}
		handleCatCommand(cluster, "nodes", "h=ip,heap.percent,ram.percent,load,node.role,master,name")
	},
}

func init() {
	EsCmd.AddCommand(nodesCmd)

	nodesCmd.Flags().StringP("cluster", "c", "localhost:9200", "es cluster")
	nodesCmd.Flags().BoolP("attrs", "a", false, "display node attributes")
}
