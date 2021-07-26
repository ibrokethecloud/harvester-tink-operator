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

	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	nodev1alpha1 "github.com/ibrokethecloud/harvester-tink-operator/api/v1alpha1"
	"github.com/ibrokethecloud/harvester-tink-operator/pkg/util"
	"github.com/pkg/errors"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=node.harvesterci.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=node.harvesterci.io,resources=clusters/status,verbs=get;update;patch

func (r *ClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("cluster", req.NamespacedName)

	// your logic here
	ok, err := util.DoesSettingExist(r.Client)
	if err != nil {
		// possible error listing the crd, requeue and retry again
		return ctrl.Result{}, err
	}

	if ok {
		// operator is installed on a harvester cluster.
		// skip reconcile of cluster objects since we will run in single cluster mode
		return ctrl.Result{}, err
	}

	clusterReq := &nodev1alpha1.Cluster{}

	if err := r.Get(ctx, req.NamespacedName, clusterReq); err != nil {
		if apierror.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch cluster")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if clusterReq.ObjectMeta.DeletionTimestamp.IsZero() {
		// reconcile the cluster objects
		var err error
		newStatus := clusterReq.Status.DeepCopy()
		switch newStatus.Status {
		case "":
			// filter and identify nodes
			newStatus, err = r.IdentifyNodes(ctx, clusterReq)
		case "ElectLeader":
			// Identify a leader if one doesnt already exist
			newStatus, err = r.ElectLeader(ctx, clusterReq)
		case "PatchNodes":
			// Patch nodes with common settings and mark them ready for provisioning
			newStatus, err = r.PatchNodes(ctx, clusterReq)
		case "Nodesubmitted":
			// Nodes have been submitted for processing
			// During watch on node objects use this for identifying
			// changes and resubmission of nodes
			r.Log.Info("processing cluster object")
			result, err := r.ReconcileNodes(ctx, clusterReq)
			return result, err
		}

		if err != nil {
			return ctrl.Result{}, err
		}
		clusterReq.Status = *newStatus
		// always requeue since the exit is via the switch flow
		return ctrl.Result{Requeue: true}, r.Update(ctx, clusterReq)
	}
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Watches the Register requests and reconciles their status
	// for remote mode ssh's into nodes and completes node reconcilliation
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodev1alpha1.Cluster{}).
		Watches(&source.Kind{Type: &nodev1alpha1.Register{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) (reconcileList []reconcile.Request) {
					labels := a.Meta.GetLabels()
					clusterName, ok := labels["clusterName"]
					if ok {
						r.Log.Info("Reconcilling registration request " + a.Meta.GetName())
						r.Log.Info("Reconcilling cluster " + clusterName)
						reconcileItem := reconcile.Request{
							NamespacedName: types.NamespacedName{
								Namespace: a.Meta.GetNamespace(),
								Name:      clusterName,
							},
						}
						reconcileList = append(reconcileList, reconcileItem)
					}
					r.Log.Info("items returned: " + fmt.Sprintf("%d", len(reconcileList)))
					return reconcileList
				}),
			}).
		Complete(r)
}

// IdentifyNodes populates the status with node details for this cluster
func (r *ClusterReconciler) IdentifyNodes(ctx context.Context, req *nodev1alpha1.Cluster) (status *nodev1alpha1.ClusterStatus, err error) {

	status = req.Status.DeepCopy()

	nodeList := &nodev1alpha1.RegisterList{}
	nodeLabels, err := labels.Parse("clusterName=" + req.Name)
	if err != nil {
		return status, err
	}

	err = r.List(ctx, nodeList, &client.ListOptions{LabelSelector: nodeLabels})

	if err != nil {
		if apierror.IsNotFound(err) {
			// request gets requeued until a node is available
			return status, nil
		} else {
			return status, err
		}
	}
	memberList := []string{}
	for _, node := range nodeList.Items {
		if !containsString(status.Members, node.Name) {
			memberList = append(memberList, node.Name)
		}
	}
	status.Members = memberList
	status.Status = "ElectLeader"

	if len(memberList) == 0 {
		status.Status = "Nodesubmitted"
	}

	return status, nil
}

func (r *ClusterReconciler) ElectLeader(ctx context.Context, req *nodev1alpha1.Cluster) (status *nodev1alpha1.ClusterStatus, err error) {
	// Patch nodes will elect a leader if one doesnt exist
	// and generate config for seed and join

	status = req.Status.DeepCopy()
	var leaderExists bool
	possibleLeaderNodes := make(map[string]*nodev1alpha1.Register)
	var leaderNode *nodev1alpha1.Register
	for _, node := range req.Status.Members {
		if !leaderExists {
			node, err := r.getNode(ctx, types.NamespacedName{Namespace: "", Name: node})
			if err != nil {
				return status, err
			}

			leader, ok := node.Labels["leader"]
			// need a node with a static address. Else ignore this node from leader election.
			if ok && leader == "true" && node.Spec.Address != "" {
				leaderExists = true
				leaderNode = node
			}

			// unset leader label
			if ok && leader == "true" && node.Spec.Address == "" {
				delete(node.Labels, "leader")
				if err := r.Update(ctx, node); err != nil {
					return status, errors.Wrap(err, "unable to unset leader label from node")
				}
			}

			if !leaderExists && node.Spec.Address != "" {
				possibleLeaderNodes[node.Name] = node
			}

		}
	}

	if !leaderExists && len(possibleLeaderNodes) != 0 {
		for _, node := range possibleLeaderNodes {
			leaderNode = node
		}
		leaderLabels := make(map[string]string)
		leaderLabels = leaderNode.GetLabels()
		leaderLabels["leader"] = "true"
		leaderNode.SetLabels(leaderLabels)
	}

	if leaderNode != nil {
		err = r.Update(ctx, leaderNode)
		if err != nil {
			return status, err
		}
	}

	// in case we are unable to find a leader //
	if !leaderExists {
		return status, fmt.Errorf("Unable to elect a leader for the cluster %s. A cluster needs a leader with a static address and label leader=true", req.Name)
	}
	status.Status = "PatchNodes"
	return status, nil
}

func (r *ClusterReconciler) PatchNodes(ctx context.Context, req *nodev1alpha1.Cluster) (status *nodev1alpha1.ClusterStatus, err error) {
	status = req.Status.DeepCopy()
	for _, node := range status.Members {
		registerReq, err := r.getNode(ctx, types.NamespacedName{Name: node, Namespace: ""})
		if err != nil {
			return status, err
		}
		// apply cluster level settings
		registerReq.Spec.Token = req.Spec.Token
		registerReq.Spec.Nameservers = req.Spec.Nameservers
		registerReq.Spec.PXEIsoURL = req.Spec.PXEIsoURL
		registerReq.Spec.SSHAuthorizedKeys = req.Spec.SSHAuthorizedKeys
		registerReq.Spec.Modules = req.Spec.Modules
		registerReq.Spec.Sysctls = req.Spec.Sysctls
		registerReq.Spec.NTPServers = req.Spec.NTPServers
		registerReq.Spec.DNSNameservers = req.Spec.DNSNameservers
		registerReq.Spec.Wifi = req.Spec.Wifi
		registerReq.Spec.Environment = req.Spec.Environment
		registerReq.Spec.Disk = req.Spec.Disk

		registerReqLabels := make(map[string]string)
		registerReqLabels = registerReq.Labels
		registerReqLabels["ready"] = "true"
		registerReq.SetLabels(registerReqLabels)

		if err = controllerutil.SetControllerReference(req, registerReq, r.Scheme); err != nil {
			return status, err
		}

		if err = controllerutil.SetOwnerReference(req, registerReq, r.Scheme); err != nil {
			return status, err
		}

		if err = r.Update(ctx, registerReq); err != nil {
			return status, err
		}
	}

	status.Status = "NodeSubmitted"

	return status, nil
}

func (r *ClusterReconciler) getNode(ctx context.Context, req types.NamespacedName) (node *nodev1alpha1.Register, err error) {
	node = &nodev1alpha1.Register{}
	err = r.Get(ctx, req, node)
	return node, err
}

func (r *ClusterReconciler) ReconcileNodes(ctx context.Context, req *nodev1alpha1.Cluster) (result ctrl.Result, err error) {
	r.Log.Info("Reconcilling cluster with additional nodes")
	currentStatus := req.Status.DeepCopy()
	newStatus, err := r.IdentifyNodes(ctx, req)
	if err != nil {
		return result, err
	}

	var additionalNodes bool
	var missingNodes bool
	var missingMembers []string
	for _, node := range newStatus.Members {
		if !containsString(currentStatus.Members, node) {
			additionalNodes = true
		}
	}

	for _, node := range currentStatus.Members {
		if !containsString(newStatus.Members, node) {
			missingNodes = true
			// delete element from the list
			missingMembers = append(missingMembers, node)
		}
	}

	for _, member := range missingMembers {
		currentStatus.Members = removeElement(currentStatus.Members, member)
	}

	if additionalNodes {
		currentStatus.Status = ""
	}

	r.Log.Info("Updating status in reconcileNodes" + currentStatus.Status)
	if additionalNodes || missingNodes {
		req.Status = *currentStatus
		err = r.Update(ctx, req)
		return result, err
	}

	return result, nil
}

func removeElement(in []string, value string) (out []string) {
	for _, element := range in {
		if element != value {
			out = append(out, element)
		}
	}

	return out
}
