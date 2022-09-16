/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package health

import (
	"context"
	"sync"
	"time"

	"github.com/ferryproxy/api/apis/traffic/v1alpha2"
	ferryversioned "github.com/ferryproxy/client-go/generated/clientset/versioned"
	healthclient "github.com/ferryproxy/ferry/pkg/health/client"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
)

type ClusterCache interface {
	ListHubs() []*v1alpha2.Hub
	GetTunnelAddressInControlPlane(hubName string) string
	UpdateHubConditions(name string, conditions []metav1.Condition) error
}

type HealthControllerConfig struct {
	Logger       logr.Logger
	Config       *restclient.Config
	ClusterCache ClusterCache
}

type HealthController struct {
	ctx            context.Context
	ferryClientset *ferryversioned.Clientset
	config         *restclient.Config
	logger         logr.Logger
	mut            sync.RWMutex
	clusterCache   ClusterCache
	cacheRoutes    []*v1alpha2.Route
	latestUpdate   time.Time
}

func NewHealthController(conf *HealthControllerConfig) *HealthController {
	return &HealthController{
		config:       conf.Config,
		clusterCache: conf.ClusterCache,
		logger:       conf.Logger,
	}
}

func (m *HealthController) Start(ctx context.Context) error {
	clientset, err := ferryversioned.NewForConfig(m.config)
	if err != nil {
		return err
	}
	m.ferryClientset = clientset
	return nil
}

func (m *HealthController) Sync(ctx context.Context) {
	m.mut.Lock()
	defer m.mut.Unlock()
	if time.Since(m.latestUpdate) <= 1*time.Second {
		return
	}
	hubs := m.clusterCache.ListHubs()

	m.check(ctx, hubs)

	m.latestUpdate = time.Now()
}

func (m *HealthController) check(ctx context.Context, hubs []*v1alpha2.Hub) {
	for _, hub := range hubs {
		host := m.clusterCache.GetTunnelAddressInControlPlane(hub.Name)
		route := healthclient.NewClient("http://" + host)
		err := route.Get(ctx)
		if err != nil {
			m.logger.Error(err, "health", "hub", hub.Name)
			err = m.clusterCache.UpdateHubConditions(hub.Name, []metav1.Condition{
				{
					Type:    v1alpha2.TunnelHealthCondition,
					Status:  metav1.ConditionTrue,
					Reason:  "Unhealth",
					Message: err.Error(),
				},
			})
		} else {
			err = m.clusterCache.UpdateHubConditions(hub.Name, []metav1.Condition{
				{
					Type:   v1alpha2.TunnelHealthCondition,
					Status: metav1.ConditionTrue,
					Reason: "Health",
				},
			})
		}
		if err != nil {
			m.logger.Error(err, "Failed update hub status", "hub", hub.Name)
		}
	}
}