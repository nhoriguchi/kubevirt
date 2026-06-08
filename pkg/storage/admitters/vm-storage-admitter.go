/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package admitters

import (
	"context"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

type Admitter struct {
	virtClient    kubecli.KubevirtClient
	ctx           context.Context
	ar            *admissionv1.AdmissionRequest
	vm            *v1.VirtualMachine
	clusterConfig *virtconfig.ClusterConfig
}

func NewAdmitter(virtClient kubecli.KubevirtClient, ctx context.Context, ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig) *Admitter {
	return &Admitter{
		virtClient:    virtClient,
		ctx:           ctx,
		ar:            ar,
		vm:            vm,
		clusterConfig: clusterConfig,
	}
}

func (a Admitter) AdmitStatus() []metav1.StatusCause {
	causes := a.validateSnapshotStatus()
	if len(causes) > 0 {
		return causes
	}

	causes = a.validateRestoreStatus()
	if len(causes) > 0 {
		return causes
	}
	return causes
}

func (a Admitter) Admit() ([]metav1.StatusCause, error) {
	causes, err := a.validateVirtualMachineDataVolumeTemplateNamespace()
	if err != nil || len(causes) > 0 {
		return causes, err
	}

	dvs, err := a.virtClient.CdiClient().CdiV1beta1().DataVolumes(a.vm.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("failed to fetch dvs: %v\n", err)
		return causes, err
	}
	fmt.Printf("-- DVS: %v\n", len(dvs.Items))
	for _, dv := range dvs.Items {
		fmt.Printf("--- name: %v\n", dv.ObjectMeta.Name)
		for _, templateDataVolume := range a.vm.Spec.DataVolumeTemplates {
			fmt.Printf("---- name2: %v\n", templateDataVolume.Name)
			if dv.ObjectMeta.Name == templateDataVolume.Name {
				fmt.Printf("----- ! DataVolume %s already exist in namespace %s", templateDataVolume.Name, a.vm.Namespace)
				causes = append(causes, metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: fmt.Sprintf("DataVolume %s already exist in namespace %s", templateDataVolume.Name, a.vm.Namespace),
				})
				return causes, nil
			}
		}
	}

	causes = a.AdmitStatus()
	if len(causes) > 0 {
		return causes, err
	}

	return causes, nil
}

func Admit(virtClient kubecli.KubevirtClient, ctx context.Context, ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig) ([]metav1.StatusCause, error) {
	storageAdmitter := NewAdmitter(virtClient, ctx, ar, vm, clusterConfig)
	return storageAdmitter.Admit()
}

func AdmitStatus(virtClient kubecli.KubevirtClient, ctx context.Context, ar *admissionv1.AdmissionRequest, vm *v1.VirtualMachine, clusterConfig *virtconfig.ClusterConfig) []metav1.StatusCause {
	storageAdmitter := NewAdmitter(virtClient, ctx, ar, vm, clusterConfig)
	return storageAdmitter.AdmitStatus()
}
