/*

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

package controllers

import (
	"context"
	"fmt"
	"math/rand"
	"net"

	"github.com/go-logr/logr"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	hcloudv1beta1 "github.com/apricote/hcloud-floating-ip-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// FloatingIPReconciler reconciles a FloatingIP object
type FloatingIPReconciler struct {
	client.Client
	Log          logr.Logger
	HCloudClient *hcloud.Client
}

// +kubebuilder:rbac:groups=hcloud.apricote.de,resources=floatingips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hcloud.apricote.de,resources=floatingips/status,verbs=get;update;patch

func (r *FloatingIPReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("floatingip", req.NamespacedName)

	// first, fetch our floating ip
	var fIP hcloudv1beta1.FloatingIP
	if err := r.Get(ctx, req.NamespacedName, &fIP); err != nil {
		// it might be not found if this is a delete request
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch floating ip1")
		return ctrl.Result{}, err
	}

	// Find Hetzner Floating IP
	ip := net.ParseIP(fIP.Spec.IP)
	if ip == nil {
		err := fmt.Errorf("error parsing ip from spec: %s", fIP.Spec.IP)
		log.Error(err, "unable to parse ip from spec")
		return ctrl.Result{}, err
	}

	fips, err := r.HCloudClient.FloatingIP.All(ctx)
	if err != nil {
		log.Error(err, "unable to fetch floating ips from hcloud")
		return ctrl.Result{}, err
	}

	var hcloudFIP *hcloud.FloatingIP
	for i := range fips {
		if fips[i].IP.Equal(ip) {
			hcloudFIP = fips[i]
		}
	}

	if hcloudFIP == nil {
		err := fmt.Errorf("ip %s does not match any hcloud floating ip resource", ip.String())
		log.Error(err, "no matching hcloud floating ip resource found")
		return ctrl.Result{}, err
	}

	// If the IP is assigned, check if node is healthy -> noop
	// TODO: Also verify that node matches NodeSelector
	if hcloudFIP.Server != nil {
		currentServer, _, err := r.HCloudClient.Server.GetByID(ctx, hcloudFIP.Server.ID)
		if err != nil {
			log.Error(err, "unable to fetch currently assigned server from hcloud")
			return ctrl.Result{}, err
		}

		var currentNode corev1.Node
		err = r.Client.Get(ctx, types.NamespacedName{
			Name: currentServer.Name,
		}, &currentNode)

		if err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err, "unable to fetch currently assigned node")
				return ctrl.Result{}, err
			}
			// If the node can not be found we will continue with assignment

		} else {
			// Node returned, verify that node is healthy
			for _, condition := range currentNode.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
					// Node is healthy, no work to
					return ctrl.Result{}, nil
				}
			}
		}
	}

	// Get all eligible nodes.
	var nodesMatchingSelector corev1.NodeList
	if err := r.List(ctx, &nodesMatchingSelector, client.MatchingLabels(fIP.Spec.NodeSelector)); err != nil {
		log.Error(err, "unable to fetch eligible nodes")
		return ctrl.Result{}, err
	}

	// Only assign to nodes with condition Ready
	var eligibleNodes []corev1.Node

	for _, node := range nodesMatchingSelector.Items {
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				eligibleNodes = append(eligibleNodes, node)
			}
		}
	}

	// Ensure that we have at least one eligible node
	total := len(eligibleNodes)
	if total == 0 {
		err := fmt.Errorf("no eligible nodes found for FloatingIP(%s)", fIP.Name)
		log.Error(err, "no eligible nodes found")
		return ctrl.Result{}, err
	}

	// To avoid running into issues where we always select a bad node, we chose
	// a random node from the list of eligble nodes.
	targetNode := eligibleNodes[rand.Intn(len(eligibleNodes)-1)]

	server, _, err := r.HCloudClient.Server.GetByName(ctx, targetNode.Name)
	if err != nil {
		log.Error(err, "unable to retrieve server from hcloud")
		return ctrl.Result{}, err
	}
	if server == nil {
		err := fmt.Errorf("server with name %s was not found in hcloud", targetNode.Name)
		return ctrl.Result{}, err
	}

	// Assign hcloud floating ip to hcloud server

	// There used to be a bug with HCloud where a floating ip stopped accepting
	// traffic when it was assigned to one node and then assigned to another node.
	// Removing the assignment first avoids this issue
	if hcloudFIP.Server != nil {
		_, _, err := r.HCloudClient.FloatingIP.Unassign(ctx, hcloudFIP)
		if err != nil {
			log.Error(err, "unable to remove floating ip assignment")
			return ctrl.Result{}, err
		}
	}

	//spew.Dump(hcloudFIP, server)
	_, _, err = r.HCloudClient.FloatingIP.Assign(ctx, hcloudFIP, server)
	if err != nil {
		log.Error(err, "unable to assign floating ip to node")
		return ctrl.Result{}, err
	}

	fIP.Status.NodeName = targetNode.Name

	if err := r.Status().Update(ctx, &fIP); err != nil {
		log.Error(err, "unable to update floating ip status")
		return ctrl.Result{}, err
	}

	log.Info("floating ip sucessfully assigned to node")
	return ctrl.Result{}, nil
}

func (r *FloatingIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hcloudv1beta1.FloatingIP{}).
		Watches(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(func(obj handler.MapObject) []ctrl.Request {
				log := r.Log.WithValues("node", obj.Meta.GetName())
				ctx := context.Background()

				var fIPList hcloudv1beta1.FloatingIPList

				// TODO ask hcloud which ips are assigned to this node
				// Current implementation just marks all floating ips
				// nodeName := obj.Meta.GetName()

				// hcloudFloatingIPs, err := r.HCloudClient.FloatingIP.All(ctx)
				// if err != nil {
				// 	log.Info("unable to get floating ips from hcloud")
				// 	return nil
				// }

				// assignedFloatingIPs

				// hcloudFloatingIPs

				if err := r.List(ctx, &fIPList, client.InNamespace(obj.Meta.GetNamespace())); err != nil {
					//spew.Dump(err)
					log.Info("unable to get floatingips for node", "node", obj)
					return nil
				}

				res := make([]ctrl.Request, len(fIPList.Items))
				for i, ip := range fIPList.Items {
					res[i].Name = ip.Name
					res[i].Namespace = ip.Namespace
				}
				return res
			}),
		}).
		Complete(r)
}
