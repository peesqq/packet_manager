package main

import (
	"fmt"
	"os"
	"path/filepath"

	"pm/internal/archive"
	"pm/internal/config"
	"pm/internal/logx"
	"pm/internal/sshclient"
)

func main() {
	if len(os.Args) < 3 {
		logx.Fatal("usage: pm [create|update] <config.(json|yaml|yml)>")
	}

	cmd := os.Args[1]
	cfgPath := os.Args[2]

	switch cmd {
	case "create":
		packet, err := config.LoadPacket(cfgPath)
		if err != nil {
			logx.Fatal("load packet: %v", err)
		}

		files, err := packet.CollectFiles()
		if err != nil {
			logx.Fatal("collect files: %v", err)
		}

		archiveName := fmt.Sprintf("%s-%s.tar.gz", packet.Name, packet.Ver)
		if packet.Output != "" {
			if err := os.MkdirAll(packet.Output, 0o755); err != nil {
				logx.Fatal("mkdir output: %v", err)
			}
			archiveName = filepath.Join(packet.Output, archiveName)
		}

		if err := archive.Create(archiveName, files); err != nil {
			logx.Fatal("create archive: %v", err)
		}
		logx.Info("archive created: %s", archiveName)

		if packet.SSH != nil {
			client, err := sshclient.New(packet.SSH)
			if err != nil {
				logx.Fatal("ssh connect: %v", err)
			}
			defer client.Close()

			if err := client.Upload(archiveName, packet.SSH.RemotePath); err != nil {
				logx.Fatal("ssh upload: %v", err)
			}
			logx.Info("uploaded to: %s", packet.SSH.RemotePath)
		}

	case "update":
		pkgs, err := config.LoadPackages(cfgPath)
		if err != nil {
			logx.Fatal("load packages: %v", err)
		}
		if pkgs.SSH == nil {
			logx.Fatal("ssh config required")
		}

		outDir := pkgs.OutputDir
		if outDir == "" {
			outDir = "./dist"
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			logx.Fatal("mkdir output: %v", err)
		}

		client, err := sshclient.New(pkgs.SSH)
		if err != nil {
			logx.Fatal("ssh connect: %v", err)
		}
		defer client.Close()

		remote := pkgs.SSH.RemotePath
		if remote == "" {
			remote = "."
		}
		matches, err := client.List(remote)
		if err != nil {
			logx.Fatal("list remote: %v", err)
		}

		selected, err := config.ResolvePackages(matches, pkgs.Packages)
		if err != nil {
			logx.Fatal("resolve packages: %v", err)
		}

		for _, sel := range selected {
			localPath := filepath.Join(outDir, filepath.Base(sel.RemotePath))
			if err := client.Download(sel.RemotePath, localPath); err != nil {
				logx.Fatal("download: %v", err)
			}
			if err := archive.Extract(localPath, outDir); err != nil {
				logx.Fatal("extract: %v", err)
			}
			logx.Info("updated: %s %s", sel.Name, sel.Version)
		}

	default:
		logx.Fatal("unknown command: %s", cmd)
	}
}
