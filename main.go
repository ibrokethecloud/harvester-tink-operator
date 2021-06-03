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

package main

import (
	"context"
	"flag"
	"os"

	web "net/http"

	"github.com/gorilla/mux"
	nodev1alpha1 "github.com/ibrokethecloud/harvester-tink-operator/api/v1alpha1"
	"github.com/ibrokethecloud/harvester-tink-operator/controllers"
	"github.com/ibrokethecloud/harvester-tink-operator/pkg/http"
	"github.com/ibrokethecloud/harvester-tink-operator/pkg/tink"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = nodev1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	config := ctrl.GetConfigOrDie()

	// Need to check CM exists //
	nonMgrClient, err := client.New(config, client.Options{})

	if err != nil {
		setupLog.Error(err, "unable to create non manager client")
	}

	fullClient, err := tink.NewClient(nonMgrClient)
	if err != nil {
		setupLog.Error(err, "unable to create tink client")
		os.Exit(1)
	}
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "9380a13a.harvesterci.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	client := mgr.GetClient()

	if err = (&controllers.RegisterReconciler{
		Client:     client,
		Log:        ctrl.Log.WithName("controllers").WithName("Register"),
		Scheme:     mgr.GetScheme(),
		FullClient: fullClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Register")
		os.Exit(1)
	}

	// api server to serve config objects
	router := mux.NewRouter()
	configServer := http.ConfigServer{
		Client: client,
		Log:    ctrl.Log.WithName("webserver").WithName("config"),
	}
	configServer.SetupRoutes(router)

	webServer := web.Server{
		Addr:    ":" + nodev1alpha1.DefaultConfigURLPort,
		Handler: router,
	}
	go func() {
		err = webServer.ListenAndServe()
		if err != nil {
			os.Exit(1)
		}
	}()

	defer func() {
		_ = webServer.Shutdown(context.Background())
	}()
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}
