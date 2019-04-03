package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
)

func init() {
	app.Commands = append(
		app.Commands,
		cli.Command{
			Name:   "sync-to-local",
			Usage:  "sync-to-local <remote source directory> <local target directory>",
			Action: SyncToLocal,
		},
		cli.Command{
			Name:   "sync-to-remote",
			Usage:  "sync-to-remote <local source directory> <remote target directory>",
			Action: SyncToRemote,
		},
	)
}

func SyncToLocal(c *cli.Context) {
	remoteDir := c.Args()[0]
	localDir := c.Args()[1]

	client, err := getSftpClient(c)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(localDir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	remoteStats := getRemoteStats(remoteDir, client)

	localStats, err := getLocalStats(localDir)
	if err != nil {
		log.Fatal(err)
	}

	filesToSync := whichFilesToSync(remoteStats, localStats)

	for _, fileToSync := range filesToSync {
		localFile := path.Join(localDir, fileToSync)
		newDir := path.Dir(localFile)
		err = os.MkdirAll(newDir, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}

		remoteFile := path.Join(remoteDir, fileToSync)
		source, err := client.Open(remoteFile)
		if err != nil {
			log.Fatalf("unable to open remote file: %s", err)
		}

		target, err := os.OpenFile(localFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if err != nil {
			log.Fatal(err)
		}

		n, err := io.Copy(target, source)
		if err != nil {
			log.Fatalf("unable to copy file: %s", err)
		}

		if n != remoteStats[fileToSync].Size() {
			log.Fatal("copied bytes size mismatch - source: %d copied: %d",
				remoteStats[fileToSync].Size(), n)
		}

		_ = source.Close()
		_ = target.Close()
		
		if err := os.Chtimes(localFile, remoteStats[fileToSync].ModTime(), remoteStats[fileToSync].ModTime()); err != nil {
			fmt.Printf("unable to set time for: %s: %s", localFile, err)
		}

		fmt.Println(fileToSync)
	}
}

func SyncToRemote(c *cli.Context) {
	localDir := c.Args()[0]
	remoteDir := c.Args()[1]

	client, err := getSftpClient(c)
	if err != nil {
		log.Fatal(err)
	}

	err = client.MkdirAll(remoteDir)
	if err != nil {
		log.Fatal(err)
	}

	remoteStats := getRemoteStats(remoteDir, client)

	localStats, err := getLocalStats(localDir)
	if err != nil {
		log.Fatal(err)
	}

	filesToSync := whichFilesToSync(localStats, remoteStats)

	for _, fileToSync := range filesToSync {
		remoteFile := path.Join(remoteDir, fileToSync)
		newDir := path.Dir(remoteFile)
		err = client.MkdirAll(newDir)
		if err != nil {
			log.Fatal(err)
		}

		localFile := path.Join(localDir, fileToSync)
		source, err := os.Open(localFile)
		if err != nil {
			log.Fatalf("unable to open local file: %s", err)
		}

		target, err := client.OpenFile(remoteFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY)
		if err != nil {
			log.Fatal(err)
		}

		n, err := io.Copy(target, source)
		if err != nil {
			log.Fatalf("unable to copy file: %s", err)
		}

		if n != localStats[fileToSync].Size() {
			log.Fatal(fmt.Sprintf("copied bytes size mismatch - %s: %d copied: %d",
				filesToSync, localStats[fileToSync].Size(), n))
		}

		_ = source.Close()
		_ = target.Close()

		if err := client.Chtimes(remoteFile, localStats[fileToSync].ModTime(), localStats[fileToSync].ModTime()); err != nil {
			fmt.Printf("unable to set time for: %s: %s", remoteFile, err)
		}

		fmt.Println(fileToSync)
	}
}

func whichFilesToSync(sourceStats, targetStats map[string]os.FileInfo) []string {
	filesToSync := make([]string, 0)

	for filename, info := range sourceStats {
		//fmt.Printf("--->>> %s -> %d %d %+v\n", filename, localStats[filename].Size(), localStats[filename].ModTime().UTC().Unix(), localStats[filename].ModTime())
		//fmt.Printf("===>>> %s -> %d %d %+v\n", filename, remoteStats[filename].Size(), remoteStats[filename].ModTime().UTC().Unix(), remoteStats[filename].ModTime())

		if targetStats[filename] == nil || targetStats[filename].Size() != info.Size() || targetStats[filename].ModTime().Unix() < info.ModTime().Unix() {
			filesToSync = append(filesToSync, filename)
		}
	}

	return filesToSync
}

func getLocalStats(localDir string) (map[string]os.FileInfo, error) {
	localStats := make(map[string]os.FileInfo)
	localDirLen := len(localDir) + 1
	// Build a list of local files
	err := filepath.Walk(localDir, func(file string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", file, err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		localStats[file[localDirLen:]] = info

		return nil
	})
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error walking the path %q: %v", localDir, err))
	}

	return localStats, nil
}

func getRemoteStats(remoteDir string, client *sftp.Client) map[string]os.FileInfo {
	remoteStats := make(map[string]os.FileInfo)
	remoteDirLen := len(remoteDir) + 1
	walker := client.Walk(remoteDir)
	for walker.Step() {
		if walker.Stat().IsDir() {
			continue
		}
		remoteStats[walker.Path()[remoteDirLen:]] = walker.Stat()
	}

	return remoteStats
}


