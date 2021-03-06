package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/thecodeteam/gocsi"
	"github.com/thecodeteam/gocsi/csi"
)

var listVolumes struct {
	maxEntries    uint32
	startingToken string
	paging        bool
}

var listVolumesCmd = &cobra.Command{
	Use:     "list-volumes",
	Aliases: []string{"ls", "list", "volumes"},
	Short:   `invokes the rpc "ListVolumes"`,
	RunE: func(*cobra.Command, []string) error {
		ctx, cancel := context.WithTimeout(root.ctx, root.timeout)
		defer cancel()

		req := csi.ListVolumesRequest{
			Version:       &root.version.Version,
			MaxEntries:    listVolumes.maxEntries,
			StartingToken: listVolumes.startingToken,
		}

		// If auto-paging is not enabled then send a normal request.
		if !listVolumes.paging {
			rep, err := controller.client.ListVolumes(ctx, &req)
			if err != nil {
				return err
			}
			return root.tpl.Execute(os.Stdout, rep)
		}

		// Paging is enabled.
		cvol, cerr := gocsi.PageAllVolumes(ctx, controller.client, req)
		for {
			select {
			case v, ok := <-cvol:
				if !ok {
					return nil
				}
				if err := root.tpl.Execute(os.Stdout, v); err != nil {
					return err
				}
			case e, ok := <-cerr:
				if !ok {
					return nil
				}
				return e
			}
		}
	},
}

func init() {
	controllerCmd.AddCommand(listVolumesCmd)

	listVolumesCmd.Flags().Uint32Var(
		&listVolumes.maxEntries,
		"max-entries",
		0,
		"the maximum number of entries to return")

	listVolumesCmd.Flags().StringVar(
		&listVolumes.startingToken,
		"starting-token",
		"",
		"the starting token used to retrieve paged data")

	listVolumesCmd.Flags().BoolVar(
		&listVolumes.paging,
		"paging",
		false,
		"a flag that enables auto-paging")

	listVolumesCmd.Flags().StringVar(
		&root.format,
		"format",
		"",
		"the go template format used to emit the results")

	listVolumesCmd.Flags().BoolVar(
		&root.withRequiresVolumeAttributes,
		"with-requires-attributes",
		false,
		withRequiresRepAttribsDesc)
}
