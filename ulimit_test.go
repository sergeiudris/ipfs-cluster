package ipfscluster

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"golang.org/x/sys/unix"
)

// we build a cluster image using it's own Dockerfile
// then we run the cluster in dokcer containers on the same network (as if docker-compose)
// then with docker exec change fd limit in cluster node(s)
// start adding/removing files (whatever needed) to recreate the error in issue#1180
// introduce the changes to code - auto increase of fd limit and env variables to turn it on/off
// now we can test before/after: with tiny fd limit it should fail, with auto increase should work
// metrics: tests should show with what parameters of fd limit and badger store happens what

func TestTinyUlimit(t *testing.T) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"echo", "hello world"},
		Tty:   false,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
}

func unixGetLimit() (uint64, uint64, error) {
	rlimit := unix.Rlimit{}
	err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rlimit)
	return rlimit.Cur, rlimit.Max, err
}

func unixSetLimit(soft uint64, max uint64) error {
	rlimit := unix.Rlimit{
		Cur: soft,
		Max: max,
	}
	return unix.Setrlimit(unix.RLIMIT_NOFILE, &rlimit)
}

func TestUlimit(t *testing.T) {

	cur, max, _ := unixGetLimit()

	fmt.Printf("cur %d, max %d \n", cur, max)

	unixSetLimit(4, 4)

	cur, max, _ = unixGetLimit()

	fmt.Printf("cur %d, max %d \n", cur, max)

}

func TestCtl(t *testing.T) {

}
