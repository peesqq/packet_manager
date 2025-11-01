package sshclient

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"pm/internal/config"
)

type Client struct {
	ssh  *ssh.Client
	sftp *sftp.Client
}

func New(cfg *config.SSHConfig) (*Client, error) {
	if cfg.Port == 0 { cfg.Port = 22 }
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port), &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{ssh.Password(cfg.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil { return nil, err }
	sc, err := sftp.NewClient(conn)
	if err != nil { conn.Close(); return nil, err }
	return &Client{ssh: conn, sftp: sc}, nil
}

func (c *Client) Close() {
	c.sftp.Close()
	c.ssh.Close()
}

func (c *Client) Upload(localPath, remoteDir string) error {
	if err := c.sftp.MkdirAll(remoteDir); err != nil { return err }
	dst := filepath.Join(remoteDir, filepath.Base(localPath))
	src, err := os.Open(localPath)
	if err != nil { return err }
	defer src.Close()
	dstFile, err := c.sftp.Create(dst)
	if err != nil { return err }
	defer dstFile.Close()
	_, err = io.Copy(dstFile, src)
	return err
}

func (c *Client) Download(remotePath, localPath string) error {
	src, err := c.sftp.Open(remotePath)
	if err != nil { return err }
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil { return err }
	dst, err := os.Create(localPath)
	if err != nil { return err }
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func (c *Client) List(remoteDir string) ([]string, error) {
	entries, err := c.sftp.ReadDir(remoteDir)
	if err != nil { return nil, err }
	var out []string
	for _, e := range entries {
		out = append(out, filepath.Join(remoteDir, e.Name()))
	}
	return out, nil
}
