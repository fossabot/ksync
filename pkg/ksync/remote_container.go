package ksync

import (
	"fmt"

	"github.com/fatih/structs"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"

	"github.com/vapor-ware/ksync/pkg/debug"
	pb "github.com/vapor-ware/ksync/pkg/proto"
)

// RemoteContainer is a specific container running on the remote cluster.
type RemoteContainer struct {
	ID       string
	Name     string
	NodeName string
	PodName  string
}

// NewRemoteContainer builds a RemoteContainer from a pod.
func NewRemoteContainer(
	pod *apiv1.Pod, containerName string) (*RemoteContainer, error) {

	// TODO: I don't want to go and setup a whole bunch of error handling code
	// right now. The NotFound error works perfectly here for now.
	if pod.DeletionTimestamp != nil {
		return nil, &errors.StatusError{
			ErrStatus: metav1.Status{
				Status:  metav1.StatusFailure,
				Reason:  metav1.StatusReasonNotFound,
				Message: fmt.Sprintf("%s scheduled for deletion", pod.Name),
			}}
	}

	// TODO: runtime error because there are no container statuses while
	// k8s master is restarting.
	// TODO: added, but non-running containers don't have any status. This should
	// be converted to something closer to `IsNotRunning()`
	// and `IsMissingContainer()`
	if containerName == "" && len(pod.Status.ContainerStatuses) > 0 {
		return &RemoteContainer{
			pod.Status.ContainerStatuses[0].ContainerID[9:],
			pod.Status.ContainerStatuses[0].Name,
			pod.Spec.NodeName,
			pod.Name}, nil
	}

	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != containerName {
			continue
		}

		return &RemoteContainer{
			status.ContainerID[9:],
			status.Name,
			pod.Spec.NodeName,
			pod.Name,
		}, nil
	}

	return nil, &errors.StatusError{
		ErrStatus: metav1.Status{
			Status:  metav1.StatusFailure,
			Reason:  metav1.StatusReasonNotFound,
			Message: fmt.Sprintf("%s not running %s", pod.Name, containerName),
		}}
}

func (c *RemoteContainer) String() string {
	return debug.YamlString(c)
}

// Fields returns a set of structured fields for logging.
func (c *RemoteContainer) Fields() log.Fields {
	return debug.StructFields(c)
}

// Message is used to serialize over gRPC
func (c *RemoteContainer) Message() (*pb.RemoteContainer, error) {
	var result pb.RemoteContainer
	if err := mapstructure.Decode(structs.Map(c), &result); err != nil {
		return nil, debug.ErrorLocation(err)
	}
	return &result, nil
}

// Radar connects to the server component (radar) and returns a client.
func (c *RemoteContainer) Radar() (pb.RadarClient, error) {
	conn, err := NewRadarInstance().RadarConnection(c.NodeName)
	if err != nil {
		return nil, debug.ErrorLocation(err)
	}
	// TODO: what's a better way to handle c?
	// defer conn.Close()

	log.WithFields(c.Fields()).Debug("radar connected")

	return pb.NewRadarClient(conn), nil
}

// RestartMirror restarts the remote mirror container responsible for this
// container.
func (c *RemoteContainer) RestartMirror() error {
	client, err := c.Radar()
	if err != nil {
		return err
	}

	if _, err := client.RestartMirror(
		context.Background(), &empty.Empty{}); err != nil {
		return debug.ErrorLocation(err)
	}

	return nil
}
