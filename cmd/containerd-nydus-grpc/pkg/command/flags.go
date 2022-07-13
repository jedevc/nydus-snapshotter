/*
 * Copyright (c) 2020. Ant Group. All rights reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package command

import (
	"os"
	"path/filepath"
	"time"

	"github.com/containerd/nydus-snapshotter/cmd/containerd-nydus-grpc/pkg/logging"
	"github.com/containerd/nydus-snapshotter/config"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

const (
	defaultAddress   = "/run/containerd-nydus/containerd-nydus-grpc.sock"
	defaultLogLevel  = logrus.InfoLevel
	defaultRootDir   = "/var/lib/containerd-nydus-grpc"
	defaultGCPeriod  = "24h"
	defaultPublicKey = "/signing/nydus-image-signing-public.key"
)

type Args struct {
	Address              string
	LogLevel             string
	LogDir               string
	ConfigPath           string
	RootDir              string
	CacheDir             string
	GCPeriod             string
	ValidateSignature    bool
	PublicKeyFile        string
	ConvertVpcRegistry   bool
	NydusdBinaryPath     string
	NydusImageBinaryPath string
	SharedDaemon         bool
	DaemonMode           string
	FsDriver             string
	SyncRemove           bool
	EnableMetrics        bool
	MetricsFile          string
	EnableStargz         bool
	DisableCacheManager  bool
	LogToStdout          bool
	EnableNydusOverlayFS bool
	NydusdThreadNum      int
	CleanupOnClose       bool
}

type Flags struct {
	Args *Args
	F    []cli.Flag
}

func buildFlags(args *Args) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "address",
			Value:       defaultAddress,
			Destination: &args.Address,
		},
		&cli.StringFlag{
			Name:        "log-level",
			Value:       defaultLogLevel.String(),
			Usage:       "set the logging level [trace, debug, info, warn, error, fatal, panic]",
			Destination: &args.LogLevel,
		},
		&cli.StringFlag{
			Name:        "log-dir",
			Value:       "",
			Usage:       "path to the log dir",
			Destination: &args.LogDir,
		},
		&cli.StringFlag{
			Name:        "config-path",
			Required:    true,
			Usage:       "path to the configuration file",
			Destination: &args.ConfigPath,
		},
		&cli.StringFlag{
			Name:        "root",
			Value:       defaultRootDir,
			Usage:       "path to the root directory for this snapshotter",
			Destination: &args.RootDir,
		},
		&cli.StringFlag{
			Name:        "cache-dir",
			Value:       "",
			Usage:       "path to the cache dir",
			Destination: &args.CacheDir,
		},
		&cli.StringFlag{
			Name:        "gc-period",
			Value:       defaultGCPeriod,
			Usage:       "period for gc blob cache, duration string(for example, 1m, 2h)",
			Destination: &args.GCPeriod,
		},
		&cli.BoolFlag{
			Name:        "validate-signature",
			Usage:       "whether force validate image bootstrap",
			Destination: &args.ValidateSignature,
		},
		&cli.StringFlag{
			Name:        "publickey-file",
			Value:       defaultPublicKey,
			Usage:       "path to publickey file of signature validation",
			Destination: &args.PublicKeyFile,
		},
		&cli.StringFlag{
			Name:        "nydusd-path",
			Value:       "",
			Usage:       "path to nydusd binary, if not set will lookup in $PATH",
			Destination: &args.NydusdBinaryPath,
		},
		&cli.StringFlag{
			Name:        "nydusimg-path",
			Value:       "",
			Usage:       "path to nydus-img binary path, if not set will lookup in $PATH",
			Destination: &args.NydusImageBinaryPath,
		},
		&cli.BoolFlag{
			Name:        "convert-vpc-registry",
			Usage:       "whether automatically convert the image to vpc registry to accelerate image pulling",
			Destination: &args.ConvertVpcRegistry,
		},
		&cli.BoolFlag{
			Name:        "shared-daemon",
			Usage:       "Deprecated, equivalent to \"--daemon-mode shared\"",
			Destination: &args.SharedDaemon,
		},
		&cli.StringFlag{
			Name:        "daemon-mode",
			Value:       config.DefaultDaemonMode,
			Usage:       "daemon mode to use, could be \"multiple\", \"shared\" or \"none\"",
			Destination: &args.DaemonMode,
		},
		&cli.StringFlag{
			Name:        "daemon-backend",
			Value:       config.FsDriverFusedev,
			Usage:       "DEPRECATED! Daemon fs backend, could be \"fusedev\", \"fscache\"",
			Destination: &args.FsDriver,
		},
		&cli.StringFlag{
			Name:        "fs-driver",
			Value:       config.FsDriverFusedev,
			Usage:       "FS device driver, could be \"fusedev\", \"fscache\"",
			Destination: &args.FsDriver,
		},
		&cli.BoolFlag{
			Name:        "sync-remove",
			Usage:       "whether to cleanup snapshots synchronously, default is asynchronous",
			Destination: &args.SyncRemove,
		},
		&cli.BoolFlag{
			Name:        "enable-metrics",
			Usage:       "whether to collect metrics",
			Destination: &args.EnableMetrics,
		},
		&cli.StringFlag{
			Name:        "metrics-file",
			Usage:       "file path to output metrics",
			Destination: &args.MetricsFile,
		},
		&cli.BoolFlag{
			Name:        "enable-stargz",
			Usage:       "whether to support stargz image (experimental)",
			Destination: &args.EnableStargz,
		},
		&cli.BoolFlag{
			Name:        "disable-cache-manager",
			Usage:       "whether to disable blob cache manager",
			Destination: &args.DisableCacheManager,
		},
		&cli.BoolFlag{
			Name:        "log-to-stdout",
			Usage:       "Print logs to standard out rather than files.",
			Destination: &args.LogToStdout,
		},
		&cli.BoolFlag{
			Name:        "enable-nydus-overlayfs",
			Usage:       "whether to enable nydus-overlayfs to mount",
			Destination: &args.EnableNydusOverlayFS,
		},
		&cli.IntFlag{
			Name:        "nydusd-thread-num",
			Usage:       "Nydusd daemon thread-num, default will be set to the number of CPUs",
			Destination: &args.NydusdThreadNum,
		},
		&cli.BoolFlag{
			Name:        "cleanup-on-close",
			Value:       false,
			Usage:       "whether to do cleanup when close snapshotter",
			Destination: &args.CleanupOnClose,
		},
	}
}

func NewFlags() *Flags {
	var args Args
	return &Flags{
		Args: &args,
		F:    buildFlags(&args),
	}
}

func Validate(args *Args, cfg *config.Config) error {
	var daemonCfg config.DaemonConfig
	if err := config.LoadConfig(args.ConfigPath, &daemonCfg); err != nil {
		return errors.Wrapf(err, "failed to load config file %q", args.ConfigPath)
	}

	if args.ValidateSignature && args.PublicKeyFile != "" {
		if _, err := os.Stat(args.PublicKeyFile); err != nil {
			return errors.Wrapf(err, "failed to find publicKey file %q", args.PublicKeyFile)
		}
	}

	if args.FsDriver == config.FsDriverFscache && args.DaemonMode != config.DaemonModeShared {
		return errors.New("file system driver `fscache` must work under `shared` daemon mode")
	}

	cfg.LogLevel = args.LogLevel
	cfg.DaemonCfg = daemonCfg
	cfg.RootDir = args.RootDir

	cfg.CacheDir = args.CacheDir
	if len(cfg.CacheDir) == 0 {
		cfg.CacheDir = filepath.Join(cfg.RootDir, "cache")
	}
	cfg.LogDir = args.LogDir
	// Always let options from CLI override those from configuration file.
	cfg.LogToStdout = args.LogToStdout
	if len(cfg.LogDir) == 0 {
		cfg.LogDir = filepath.Join(cfg.RootDir, logging.DefaultLogDirName)
	}
	cfg.ValidateSignature = args.ValidateSignature
	cfg.PublicKeyFile = args.PublicKeyFile
	cfg.ConvertVpcRegistry = args.ConvertVpcRegistry
	cfg.Address = args.Address
	cfg.NydusdBinaryPath = args.NydusdBinaryPath
	cfg.NydusImageBinaryPath = args.NydusImageBinaryPath
	cfg.DaemonMode = args.DaemonMode
	// Give --shared-daemon higher priority
	if args.SharedDaemon {
		cfg.DaemonMode = config.DaemonModeShared
	}
	cfg.SyncRemove = args.SyncRemove
	cfg.EnableMetrics = args.EnableMetrics
	cfg.MetricsFile = args.MetricsFile
	cfg.EnableStargz = args.EnableStargz
	cfg.DisableCacheManager = args.DisableCacheManager
	cfg.EnableNydusOverlayFS = args.EnableNydusOverlayFS
	cfg.NydusdThreadNum = args.NydusdThreadNum
	cfg.CleanupOnClose = args.CleanupOnClose
	cfg.FsDriver = args.FsDriver

	d, err := time.ParseDuration(args.GCPeriod)
	if err != nil {
		return errors.Wrapf(err, "parse gc period %v failed", args.GCPeriod)
	}
	cfg.GCPeriod = d
	return cfg.SetupNydusBinaryPaths()
}
