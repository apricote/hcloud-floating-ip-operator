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
	"net"

	"github.com/go-logr/logr"
	"github.com/hetznercloud/hcloud-go/hcloud"
	"k8s.io/apimachinery/pkg/api/errors"
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

	// Get all eligible targets.
	var eligibleNodes corev1.NodeList
	if err := r.List(ctx, &eligibleNodes, client.MatchingLabels(fIP.Spec.NodeSelector)); err != nil {
		log.Error(err, "unable to fetch eligible nodes")
		return ctrl.Result{}, err
	}

	// TODO check node status

	// Ensure that we have at least one eligible node
	total := len(eligibleNodes.Items)
	if total == 0 {
		err := fmt.Errorf("no eligible nodes found for FloatingIP(%s)", fIP.Name)
		log.Error(err, "no eligible nodes found")
		return ctrl.Result{}, err
	}

	// TODO: verify that reassignment is absolutly necessary

	// In pratice it should not make a difference which node we choose
	// as long as the NodeSelector is properly set up.
	// Might as well choose the first item of the list
	targetNode := eligibleNodes.Items[0]

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

	// Find HCloud server mathing selected node
	server, _, err := r.HCloudClient.Server.GetByName(ctx, targetNode.Name)
	if err != nil {
		log.Error(err, "selected node not found in hcloud server list")
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

	_, _, err = r.HCloudClient.FloatingIP.Assign(ctx, hcloudFIP, server)
	if err != nil {
		log.Error(err, "unable to assign floating ip to ")
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
				var floatingips hcloudv1beta1.FloatingIPList
				if err := r.List(context.Background(), &floatingips, client.InNamespace(obj.Meta.GetNamespace()), client.MatchingField(".status.nodeName", obj.Meta.GetName())); err != nil {
					r.Log.Info("unable to get floatingips for node", "node", obj)
					return nil
				}

				res := make([]ctrl.Request, len(floatingips.Items))
				for i, ip := range floatingips.Items {
					res[i].Name = ip.Name
					res[i].Namespace = ip.Namespace
				}
				return res
			}),
		}).
		Complete(r)
}
