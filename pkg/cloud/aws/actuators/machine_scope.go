// Copyright © 2018 The Kubernetes Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package actuators

import (
	"github.com/pkg/errors"
	"k8s.io/klog"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/apis/awsprovider/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	client "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset/typed/cluster/v1alpha1"
)

// MachineScopeParams defines the input parameters used to create a new MachineScope.
type MachineScopeParams struct {
	AWSClients
	Cluster *clusterv1.Cluster
	Machine *clusterv1.Machine
	Client  client.ClusterV1alpha1Interface
}

// NewMachineScope creates a new MachineScope from the supplied parameters.
// This is meant to be called for each machine actuator operation.
func NewMachineScope(params MachineScopeParams) (*MachineScope, error) {
	scope, err := NewScope(ScopeParams{AWSClients: params.AWSClients, Client: params.Client, Cluster: params.Cluster})
	if err != nil {
		return nil, err
	}

	machineConfig, err := v1alpha1.MachineConfigFromProviderConfig(params.Machine.Spec.ProviderConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get machine config")
	}

	machineStatus, err := v1alpha1.MachineStatusFromProviderStatus(params.Machine.Status.ProviderStatus)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get machine provider status")
	}

	var machineClient client.MachineInterface
	if params.Client != nil {
		machineClient = params.Client.Machines(params.Machine.Namespace)
	}

	return &MachineScope{
		Scope:         scope,
		Machine:       params.Machine,
		MachineClient: machineClient,
		MachineConfig: machineConfig,
		MachineStatus: machineStatus,
	}, nil
}

// MachineScope defines a scope defined around a machine and its cluster.
type MachineScope struct {
	*Scope

	Machine       *clusterv1.Machine
	MachineClient client.MachineInterface
	MachineConfig *v1alpha1.AWSMachineProviderConfig
	MachineStatus *v1alpha1.AWSMachineProviderStatus
}

func (m *MachineScope) storeMachineStatus() error {
	ext, err := v1alpha1.EncodeMachineStatus(m.MachineStatus)
	if err != nil {
		return err
	}

	m.Machine.Status.ProviderStatus = ext

	if _, err := m.MachineClient.UpdateStatus(m.Machine); err != nil {
		return err
	}

	return nil
}

func (m *MachineScope) Close() {
	defer m.Scope.Close()

	if m.MachineClient == nil {
		return
	}

	if _, err := m.MachineClient.Update(m.Machine); err != nil {
		klog.Errorf("[machinescope] failed to update machine: %v", err)
	}

	if err := m.storeMachineStatus(); err != nil {
		klog.Errorf("[machinescope] failed to store provider status for machine %q in namespace %q: %v", m.Machine.Name, m.Machine.Namespace, err)
	}
}
