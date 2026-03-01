package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/bziks/gitlab-package-finder/cmd/api"
	"github.com/bziks/gitlab-package-finder/cmd/migrate"
	"github.com/bziks/gitlab-package-finder/cmd/projectssync"
	"github.com/bziks/gitlab-package-finder/cmd/searchprocessing"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := cobra.Command{}

	cmd.AddCommand(
		api.Command(),
		migrate.Command(),
		projectssync.Command(),
		searchprocessing.Command(),
	)

	if err := cmd.ExecuteContext(ctx); err != nil {
		panic(err)
	}
}
