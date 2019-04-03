package main

import (
	"github.com/codegangsta/cli"
	"github.com/pkg/sftp"
	_ "github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
)

var app = cli.NewApp()

func main() {
	app.Name = "sinker"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "host",
			Value: "localhost",
		},
		cli.StringFlag{
			Name: "user",
		},
		cli.StringFlag{
			Name: "key",
		},
	}

	app.Run(os.Args)
}

func getSftpClient(c *cli.Context) (*sftp.Client, error){
	remoteHost := c.GlobalString("host")
	key := c.GlobalString("key")
	user := c.GlobalString("user")

	config, err := privateKey(user, key, ssh.InsecureIgnoreHostKey())
	if err != nil {
		log.Fatalf("unable to get private key: %s", err)
	}

	conn, err := ssh.Dial("tcp", remoteHost+":22", config)
	if err != nil {
		log.Fatalf("unable to connect to remote host '%s': %s", remoteHost, err)
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func privateKey(username string, path string, keyCallBack ssh.HostKeyCallback) (*ssh.ClientConfig, error) {
	privateKey, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(privateKey)

	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: keyCallBack,
	}, nil
}
