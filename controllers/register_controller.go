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
	"encoding/json"
	"strings"

	"github.com/tinkerbell/tink/protos/hardware"

	"github.com/ibrokethecloud/harvester-tink-operator/pkg/tink"
	"github.com/ibrokethecloud/harvester-tink-operator/pkg/util"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	nodev1alpha1 "github.com/ibrokethecloud/harvester-tink-operator/api/v1alpha1"
	"github.com/pkg/errors"
	hw "github.com/tinkerbell/tink/client"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	regoFinalizer = "register.harvesterci.io"
	UIDGenerated  = "uidgenerated"
	HWPushed      = "hardwarepushed"
)

// RegisterReconciler reconciles a Register object
type RegisterReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	FullClient *hw.FullClient
}

// +kubebuilder:rbac:groups=node.harvesterci.io,resources=registers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=node.harvesterci.io,resources=registers/status,verbs=get;update;patch

func (r *RegisterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("register", req.NamespacedName)

	regoReq := &nodev1alpha1.Register{}

	if err := r.Get(ctx, req.NamespacedName, regoReq); err != nil {
		if apierror.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch instance")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if regoReq.ObjectMeta.DeletionTimestamp.IsZero() {
		// reconile object
		var err error
		newStatus := &nodev1alpha1.RegisterStatus{}

		switch regoReq.Status.DeepCopy().Status {
		case "":
			// create uuid
			newStatus, err = r.generateUID(regoReq)
		case UIDGenerated:
			// make hardware call
			newStatus, err = r.generateHardware(ctx, regoReq)
		case HWPushed:
			return ctrl.Result{}, nil
		}
		regoReq.Status = *newStatus
		controllerutil.AddFinalizer(regoReq, regoFinalizer)
		if err != nil {
			return ctrl.Result{}, err
		}
		// always requeue since we want it to exit reconile loop via the switch flow //
		return ctrl.Result{Requeue: true}, r.Update(ctx, regoReq)
	} else {
		if containsString(regoReq.ObjectMeta.Finalizers, regoFinalizer) {
			if len(regoReq.Status.UUID) != 0 {
				if err := r.deleteHardware(ctx, regoReq.Status.UUID); err != nil {
					return ctrl.Result{}, err
				}
			}

		}
		controllerutil.RemoveFinalizer(regoReq, regoFinalizer)
		if err := r.Update(ctx, regoReq); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *RegisterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodev1alpha1.Register{}).
		Complete(r)
}

// containsString is a helper to check if finalizer exists
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// generate uid will only generate a random uid and update the object in prep for hardware push call //
func (r *RegisterReconciler) generateUID(regoReq *nodev1alpha1.Register) (regoStatus *nodev1alpha1.RegisterStatus, err error) {
	regoStatus = regoReq.Status.DeepCopy()
	labels := regoReq.GetLabels()
	if labels != nil {
		regoID, ok := labels["uuid"]
		if ok {
			regoStatus.UUID = regoID
			return regoStatus, nil
		}
	} else {
		labels = make(map[string]string)
	}

	uuid := uuid.New().String()
	labels["uuid"] = uuid
	regoReq.Labels = labels
	regoStatus.UUID = uuid
	regoStatus.Status = UIDGenerated
	return regoStatus, nil
}

// generate the tink hardware request and perform a push operation //
func (r *RegisterReconciler) generateHardware(ctx context.Context, regoReq *nodev1alpha1.Register) (regoStatus *nodev1alpha1.RegisterStatus, err error) {

	regoStatus = regoReq.Status.DeepCopy()

	regoURL, err := util.FetchServerURL(r.Client)
	if err != nil {
		return regoStatus, errors.Wrap(err, "error fetching server url")
	}

	hwRequest, err := tink.GenerateHWRequest(regoReq, regoURL)
	if err != nil {
		return regoStatus, errors.Wrap(err, "error during generatehwrequest")
	}

	hwByte, err := json.Marshal(hwRequest)
	if err != nil {
		return regoStatus, errors.Wrap(err, "error during hw request marshal")
	}

	r.Log.Info(string(hwByte))
	_, err = r.FullClient.HardwareClient.Push(ctx, &hardware.PushRequest{Data: hwRequest})
	if err != nil {
		return regoStatus, errors.Wrap(err, "error during hardware push")
	}

	regoStatus.Status = HWPushed
	return regoStatus, nil
}

func (r *RegisterReconciler) deleteHardware(ctx context.Context, uuid string) (err error) {
	_, err = r.getHardware(ctx, uuid)
	if err != nil {
		if strings.Contains(err.Error(), "sql: no rows in result set") {
			return nil
		} else {
			return errors.Wrap(err, "error during get hardware")
		}
	}

	_, err = r.FullClient.HardwareClient.Delete(ctx, &hardware.DeleteRequest{Id: uuid})
	return err
}

func (r *RegisterReconciler) getHardware(ctx context.Context, uuid string) (hw *hardware.Hardware, err error) {
	hw, err = r.FullClient.HardwareClient.ByID(ctx, &hardware.GetRequest{Id: uuid})
	return hw, err
}
