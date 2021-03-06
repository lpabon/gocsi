package cmd

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/thecodeteam/gocsi/csi"
)

var nodeUnpublishVolume struct {
	nodeID     string
	targetPath string
}

var nodeUnpublishVolumeCmd = &cobra.Command{
	Use:     "unpublish",
	Aliases: []string{"umount", "unmount"},
	Short:   `invokes the rpc "NodeUnpublishVolume"`,
	Example: `
USAGE

    csc node unpublishvolume [flags] VOLUME_ID [VOLUME_ID...]
`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		req := csi.NodeUnpublishVolumeRequest{
			Version:         &root.version.Version,
			TargetPath:      nodeUnpublishVolume.targetPath,
			UserCredentials: root.userCreds,
		}

		for i := range args {
			ctx, cancel := context.WithTimeout(root.ctx, root.timeout)
			defer cancel()

			// Set the volume ID for the current request.
			req.VolumeId = args[i]

			log.WithField("request", req).Debug("mounting volume")
			_, err := node.client.NodeUnpublishVolume(ctx, &req)
			if err != nil {
				return err
			}

			fmt.Println(args[i])
		}

		return nil
	},
}

func init() {
	nodeCmd.AddCommand(nodeUnpublishVolumeCmd)

	nodeUnpublishVolumeCmd.Flags().StringVar(
		&nodeUnpublishVolume.targetPath,
		"target-path",
		"",
		"The path from which to unmount the volume")

	nodeUnpublishVolumeCmd.Flags().BoolVar(
		&root.withRequiresCreds,
		"with-requires-credentials",
		false,
		withRequiresCredsDesc)
}
