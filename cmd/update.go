package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func makeUpdate() *cobra.Command {
	var command = &cobra.Command{
		Use:          "update",
		Short:        "Print update instructions",
		Example:      `  inletsctl update`,
		SilenceUsage: false,
	}
	command.Run = func(cmd *cobra.Command, args []string) {
		fmt.Println(updateStr)
	}
	return command
}

const updateStr = `You can update inletsctl with the following:

# For Linux/MacOS:
curl -SLfs https://inletsctl.inlets.dev | sudo sh

# For Windows (using Git Bash)
curl -SLfs https://inletsctl.inlets.dev | sh

# Or download from GitHub: https://github.com/inlets/inletsctl/releases

Thanks for using inletsctl!`
