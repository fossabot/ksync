package main

import (
	"os"
	"runtime"
	"text/template"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"github.com/vapor-ware/ksync/pkg/cli"
	"github.com/vapor-ware/ksync/pkg/ksync"
	pb "github.com/vapor-ware/ksync/pkg/proto"
	"github.com/vapor-ware/ksync/pkg/radar"
)

type versionCmd struct {
	cli.BaseCmd
}

func (v *versionCmd) new() *cobra.Command {
	long := `Print version information.`
	example := ``

	v.Init("ksync", &cobra.Command{
		Use:     "version",
		Short:   "Print version information.",
		Long:    long,
		Example: example,
		Run:     v.run,
	})

	return v.Cmd
}

var ksyncVersionTemplate = `{{define "ksync"}}ksync:
	Version:    {{.Client.Version}}
	Go Version: {{.Client.GoVersion}}
	Git Commit: {{.Client.GitCommit}}
	Git Tag:    {{if ne .Client.GitTag ""}}{{.Client.GitTag}}{{end}}
	Built:      {{.Client.BuildDate}}
	OS/Arch:    {{.Client.OS}}/{{.Client.Arch}}{{println}}{{end}}`

var radarVersionTemplate = `{{define "radar"}}radar:
	Version:    {{.Server.Version}}
	Go Version: {{.Server.GoVersion}}
	Git Commit: {{.Server.GitCommit}}
	Git Tag:    {{if ne .Server.GitTag ""}}{{.Server.GitTag}}{{end}}
	Built:      {{.Server.BuildDate}}
	Healthy:    {{.Server.Healthy}}{{println}}{{end}}`

type versionInfo struct {
	Client ksync.Version
	Server radar.Version
}

func (v *versionCmd) run(cmd *cobra.Command, args []string) { // nolint: gocyclo
	template, err := template.New("ksync").Parse(ksyncVersionTemplate)
	if err != nil {
		log.Fatal(err)
	}
	template, err = template.New("radar").Parse(radarVersionTemplate)
	if err != nil {
		log.Fatal(err)
	}

	// Get version info from radar running remotely
	// TODO: Lots of clean up. I don't like the way this works.
	var radarVersion *pb.VersionInfo
	if radarCheck() {
		radarInstance := ksync.NewRadarInstance()
		nodes, err := radarInstance.NodeNames() // nolint: vetshadow
		if err != nil {
			log.Fatal(err)
		}
		connection, err := radarInstance.RadarConnection(nodes[0])
		if err != nil {
			log.Fatal(err)
		}
		radarVersion, err = pb.NewRadarClient(
			connection).GetVersionInfo(context.Background(), &empty.Empty{})
		if err != nil {
			log.Fatal(err)
		}
	}

	version := versionInfo{
		Client: ksync.Version{
			Version:   ksync.VersionString,
			GoVersion: ksync.GoVersion,
			GitCommit: ksync.GitCommit,
			GitTag:    ksync.GitTag,
			BuildDate: ksync.BuildDate,
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		},
		Server: radar.Version{
			Version:   radarVersion.Version,
			GoVersion: radarVersion.GoVersion,
			GitCommit: radarVersion.GitCommit,
			GitTag:    radarVersion.GitTag,
			BuildDate: radarVersion.BuildDate,
			Healthy:   radarCheck(),
		},
	}

	// Convert time to a human readable format
	timeKsync, timeErr := time.Parse(time.RFC3339Nano, version.Client.BuildDate)
	if timeErr == nil {
		version.Client.BuildDate = timeKsync.Format(time.UnixDate)
	} else {
		log.Fatal(timeErr)
	}

	timeRadar, timeErr := time.Parse(time.RFC3339, version.Server.BuildDate)
	if timeErr == nil {
		version.Server.BuildDate = timeRadar.Format(time.UnixDate)
	} else {
		log.Fatal(timeErr)
	}

	// If radar is reachable, print that part of the template
	err = template.ExecuteTemplate(os.Stdout, "ksync", version)
	if err != nil {
		log.Fatal(err)
	}

	if radarCheck() {
		err := template.ExecuteTemplate(os.Stdout, "radar", version)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// TODO: temporary
func radarCheck() bool {
	radar := ksync.NewRadarInstance()
	nodes, err := radar.NodeNames()
	if err != nil {
		return false
	}

	if len(nodes) == 0 {
		return false
	}

	health, err := radar.IsHealthy(nodes[0])
	if err != nil {
		return false
	}

	return health
}
