package endpoint

import (
	"context"
	pluginapi "github.com/kubewharf/katalyst-api/pkg/protocol/evictionplugin/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

type dummyRecover struct {
	Endpoint
}

func (d *dummyRecover) GetTopEvictionPods(c context.Context, request *pluginapi.GetTopEvictionPodsRequest) (*pluginapi.GetTopEvictionPodsResponse, error) {
	return &pluginapi.GetTopEvictionPodsResponse{
		TargetPods: []*v1.Pod{{
			ObjectMeta: metav1.ObjectMeta{
				UID: "123321",
			},
		}},
		DeletionOptions: nil,
	}, nil
}

func (d *dummyRecover) Recover() (Endpoint, error) {
	return d, nil
}

func TestRecoverableEndpoint_GetTopEvictionPods(t *testing.T) {
	t.Parallel()

	rep := RecoverableEndpoint{
		pluginName: "fake-reco-ep",
		recover:    &dummyRecover{},
	}

	resp, err := rep.GetTopEvictionPods(context.TODO(), &pluginapi.GetTopEvictionPodsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, 1, len(resp.TargetPods))
	assert.Equal(t, "123321", string(resp.TargetPods[0].UID))
}

type dummyUnrecover struct{}

func (d *dummyUnrecover) Recover() (Endpoint, error) {
	return nil, errors.New("test unrecoverable")
}

func TestRecoverableEndpoint_GetTopEvictionPods_Unrecovered(t *testing.T) {
	t.Parallel()

	rep := RecoverableEndpoint{
		pluginName: "fake-reco-ep",
		recover:    &dummyUnrecover{},
	}

	resp, err := rep.GetTopEvictionPods(context.TODO(), &pluginapi.GetTopEvictionPodsRequest{})

	assert.NoError(t, err)
	assert.Equal(t, 0, len(resp.TargetPods))
}

type dummyFaultyRecover struct {
	Endpoint
}

func (d *dummyFaultyRecover) GetTopEvictionPods(c context.Context, request *pluginapi.GetTopEvictionPodsRequest) (*pluginapi.GetTopEvictionPodsResponse, error) {
	return nil, errors.New("test endpoint error")
}

func (d *dummyFaultyRecover) Recover() (Endpoint, error) {
	return d, nil
}

func TestRecoverableEndpoint_GetTopEvictionPods_FaultyReturn(t *testing.T) {
	t.Parallel()

	rep := RecoverableEndpoint{
		pluginName: "fake-reco-ep",
		recover:    &dummyFaultyRecover{},
	}

	_, err := rep.GetTopEvictionPods(context.TODO(), &pluginapi.GetTopEvictionPodsRequest{})

	assert.Error(t, err)
	assert.Equal(t, "test endpoint error", err.Error())
	assert.Nilf(t, rep.endpoint, "faulty method call should clear endpoint connection")
}
