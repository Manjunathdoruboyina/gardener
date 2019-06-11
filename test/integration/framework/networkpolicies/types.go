// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package networkpolicies

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
)

// SourcePod holds the data about pods in the shoot namespace and their services.
type SourcePod struct {
	Pod
	Ports            []Port
	ExpectedPolicies sets.String
}

// TargetPod contains data about a Pod listening on a specific port.
type TargetPod struct {
	Pod
	Port
}

// Pod contains the barebone detals about a Pod.
type Pod struct {
	Name                   string
	Labels                 labels.Set
	ShootVersionConstraint string
}

// Port holds the data about a single port.
type Port struct {
	Port int32
	Name string
}

// HostRule contains a target Host and decision if it's visible to the source Pod.
type HostRule struct {
	Host
	Allowed bool
}

// PodRule contains a rule which allows/disallow traffic to a TargetPod.
type PodRule struct {
	TargetPod
	Allowed bool
}

// Host containts host with port and optional description.
type Host struct {
	Description string
	HostName    string
	Port        int32
}

// Rule contains Pod and target Pods and Hosts to which it's (not) allowed to talk to.
type Rule struct {
	*SourcePod
	TargetPods  []PodRule
	TargetHosts []HostRule
}

// CloudAwarePodInfo contains a Cloud-specific information for Source(s) to Target(s) communication.
type CloudAwarePodInfo interface {
	// ToSources returns a list of all sources of the CloudProvider.
	ToSources() []Rule

	// EgressFromOtherNamespaces returns a list of all TargetPod.
	EgressFromOtherNamespaces(source *SourcePod) Rule

	// Provider returns the CloudProvider.
	Provider() v1beta1.CloudProvider
}

// NewPod creates a new instance of Pod.
func NewPod(name string, labels labels.Set, shootVersionContstraints ...string) Pod {
	constraint := ""
	if len(shootVersionContstraints) > 0 {
		constraint = shootVersionContstraints[0]
	}
	return Pod{name, labels, constraint}
}

// ToString returns the string represetnation of TargetHost.
func (t *HostRule) ToString() string {
	action := "block"
	if t.Allowed {
		action = "allow"
	}
	return fmt.Sprintf("should %s connection to %q %s:%d", action, t.Host.Description, t.Host.HostName, t.Host.Port)
}

// ToString returns the string represetnation of TargetPod.
func (p *PodRule) ToString() string {
	action := "block"
	if p.Allowed {
		action = "allow"
	}
	return fmt.Sprintf("should %s connection to Pod %q at port %d", action, p.TargetPod.Pod.Name, p.TargetPod.Port.Port)
}

// NewSinglePort returns just one port.
func NewSinglePort(p int32) []Port {
	return []Port{{Port: p}}
}

// CheckVersion checks if shoot version  is matched by ShootVersionConstraint.
func (p *Pod) CheckVersion(shoot *v1beta1.Shoot) bool {
	if len(p.ShootVersionConstraint) == 0 {
		return true
	}
	c, err := semver.NewConstraint(p.ShootVersionConstraint)
	if err != nil {
		panic(fmt.Sprintf("Error parsing Pod Version contstraint for pod %v: %v", *p, err))
	}
	v, err := semver.NewVersion(shoot.Spec.Kubernetes.Version)
	if err != nil {
		panic(fmt.Sprintf("Error parsing version %v", err))
	}
	return c.Check(v)
}

// Selector returns label selector for specific pod.
func (p *Pod) Selector() labels.Selector {
	return labels.SelectorFromSet(p.Labels)
}

// AsTargetPods returns a list of TargetPods for each Port.
// Returned slice is not deep copied!
func (s *SourcePod) AsTargetPods() []*TargetPod {

	targetPods := []*TargetPod{}
	for _, port := range s.Ports {
		targetPods = append(targetPods, &TargetPod{
			Pod:  s.Pod,
			Port: port,
		})
	}
	return targetPods
}

// FromPort returns a TargetPod containing only one specific port.
// This resource is not deep copied!
func (s *SourcePod) FromPort(portName string) *TargetPod {
	for _, port := range s.Ports {
		if port.Name == portName {
			return &TargetPod{
				Pod:  s.Pod,
				Port: port,
			}
		}
	}
	panic(fmt.Sprintf("Port named %q not found", portName))
}

// DummyPort returns a TargetPod containing only one 8080 port.
// This resource is not deep copied!
func (s *SourcePod) DummyPort() *TargetPod {
	if len(s.Ports) > 0 {
		panic(fmt.Sprintf("DummyPort should only be used for Pods without a Port"))
	}
	return &TargetPod{
		Pod:  s.Pod,
		Port: Port{Port: 8080, Name: "dummy"},
	}
}

// NamespacedSourcePod holds namespaced PodInfo.
type NamespacedSourcePod struct {
	*SourcePod
	Namespace string
}

// NewNamespacedSourcePod creates a new NamespacedSourcePod.
func NewNamespacedSourcePod(sp *SourcePod, namespace string) *NamespacedSourcePod {
	return &NamespacedSourcePod{SourcePod: sp, Namespace: namespace}
}

// NamespacedTargetPod holds namespaced TargetPod.
type NamespacedTargetPod struct {
	*TargetPod
	Namespace string
}

// NewNamespacedTargetPod creates a new NamespacedTargetPod.
func NewNamespacedTargetPod(tp *TargetPod, namespace string) *NamespacedTargetPod {
	return &NamespacedTargetPod{TargetPod: tp, Namespace: namespace}
}