package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/net/context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	//cli, err := client.NewEnvClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cli, err := client.NewClient("unix:///var/run/docker.sock", "1.24", nil, nil)
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
		reader, err := cli.ContainerLogs(ctx, container.ID[:10], types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(os.Stdout, reader)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		stat, err := cli.ContainerStats(ctx, container.ID[:10], false)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(os.Stdout, stat.Body)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
	}
}
