package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openserbia/doclint/pkg/cache"
	"github.com/openserbia/doclint/pkg/config"
)

func newCacheCmd() *cobra.Command {
	var cacheDir string
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the lint result cache",
	}
	cmd.PersistentFlags().StringVar(&cacheDir, "cache-dir", "", "cache directory (default: per-user cache dir)")

	clean := &cobra.Command{
		Use:   "clean",
		Short: "Delete the cached lint results",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir := resolveCacheDir(cacheDir)
			if dir == "" {
				return errors.New("could not resolve the cache directory; pass --cache-dir")
			}
			if err := cache.Open(dir).Clean(); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "cache cleaned")
			return err
		},
	}
	status := &cobra.Command{
		Use:   "status",
		Short: "Show the cache location and number of cached files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir := resolveCacheDir(cacheDir)
			if dir == "" {
				return errors.New("could not resolve the cache directory; pass --cache-dir")
			}
			c := cache.Open(dir)
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "dir:     %s\nentries: %d\n", dir, c.Entries())
			return err
		},
	}
	cmd.AddCommand(clean, status)
	return cmd
}

// resolveCacheDir returns the explicit --cache-dir when set, else the per-user
// default cache dir; "" when neither can be determined.
func resolveCacheDir(flag string) string {
	if flag != "" {
		return flag
	}
	dir, err := cache.DefaultDir()
	if err != nil {
		return ""
	}
	return dir
}

// configHash is a stable digest of the resolved config, so editing .doclint.yaml
// invalidates cached findings.
func configHash(cfg *config.Config) string {
	b, err := json.Marshal(cfg)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
