package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"
)

// -----------------------------------------------------------------------------
// Main
// -----------------------------------------------------------------------------

const (
	outputFile = "controllers/configuration/zz_generated_controllers.go"

	corev1     = "k8s.io/api/core/v1"
	netv1      = "k8s.io/api/networking/v1"
	netv1beta1 = "k8s.io/api/networking/v1beta1"
	extv1beta1 = "k8s.io/api/extensions/v1beta1"

	kongv1          = "github.com/kong/kubernetes-ingress-controller/railgun/apis/configuration/v1"
	kongv1alpha1    = "github.com/kong/kubernetes-ingress-controller/railgun/apis/configuration/v1alpha1"
	kongv1beta1     = "github.com/kong/kubernetes-ingress-controller/railgun/api/configuration/v1beta1"
	knativev1alpha1 = "knative.dev/networking/pkg/apis/networking/v1alpha1"
)

// inputControllersNeeded is a list of the supported Types for the
// Kong Kubernetes Ingress Controller. If you need to add a new type
// for support, add it here and a new controller will be generated
// when you run `make controllers`.
var inputControllersNeeded = &typesNeeded{
	typeNeeded{
		PackageImportAlias:                "corev1",
		PackageAlias:                      "CoreV1",
		Package:                           corev1,
		Type:                              "Service",
		Plural:                            "services",
		URL:                               "\"\"",
		CacheType:                         "Service",
		AcceptsIngressClassNameAnnotation: false,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "corev1",
		PackageAlias:                      "CoreV1",
		Package:                           corev1,
		Type:                              "Endpoints",
		Plural:                            "endpoints",
		URL:                               "\"\"",
		CacheType:                         "Endpoint",
		AcceptsIngressClassNameAnnotation: false,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "netv1",
		PackageAlias:                      "NetV1",
		Package:                           netv1,
		Type:                              "Ingress",
		Plural:                            "ingresses",
		URL:                               "networking.k8s.io",
		CacheType:                         "IngressV1",
		AcceptsIngressClassNameAnnotation: true,
		AcceptsIngressClassNameSpec:       true,
	},
	typeNeeded{
		PackageImportAlias:                "netv1beta1",
		PackageAlias:                      "NetV1Beta1",
		Package:                           netv1beta1,
		Type:                              "Ingress",
		Plural:                            "ingresses",
		URL:                               "networking.k8s.io",
		CacheType:                         "IngressV1beta1",
		AcceptsIngressClassNameAnnotation: true,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "extv1beta1",
		PackageAlias:                      "ExtV1Beta1",
		Package:                           extv1beta1,
		Type:                              "Ingress",
		Plural:                            "ingresses",
		URL:                               "apiextensions.k8s.io",
		CacheType:                         "IngressV1beta1",
		AcceptsIngressClassNameAnnotation: true,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "kongv1",
		PackageAlias:                      "KongV1",
		Package:                           kongv1,
		Type:                              "KongIngress",
		Plural:                            "kongingresses",
		URL:                               "configuration.konghq.com",
		CacheType:                         "KongIngress",
		AcceptsIngressClassNameAnnotation: false,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "kongv1",
		PackageAlias:                      "KongV1",
		Package:                           kongv1,
		Type:                              "KongPlugin",
		Plural:                            "kongplugins",
		URL:                               "configuration.konghq.com",
		CacheType:                         "Plugin",
		AcceptsIngressClassNameAnnotation: false,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "kongv1",
		PackageAlias:                      "KongV1",
		Package:                           kongv1,
		Type:                              "KongClusterPlugin",
		Plural:                            "kongclusterplugins",
		URL:                               "configuration.konghq.com",
		CacheType:                         "ClusterPlugin",
		AcceptsIngressClassNameAnnotation: true,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "kongv1",
		PackageAlias:                      "KongV1",
		Package:                           kongv1,
		Type:                              "KongConsumer",
		Plural:                            "kongconsumers",
		URL:                               "configuration.konghq.com",
		CacheType:                         "Consumer",
		AcceptsIngressClassNameAnnotation: true,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "kongv1alpha1",
		PackageAlias:                      "KongV1Alpha1",
		Package:                           kongv1alpha1,
		Type:                              "UDPIngress",
		Plural:                            "udpingresses",
		URL:                               "configuration.konghq.com",
		CacheType:                         "UDPIngress",
		AcceptsIngressClassNameAnnotation: true,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "kongv1beta1",
		PackageAlias:                      "KongV1Beta1",
		Package:                           kongv1beta1,
		Type:                              "TCPIngress",
		Plural:                            "tcpingresses",
		URL:                               "configuration.konghq.com",
		CacheType:                         "TCPIngress",
		AcceptsIngressClassNameAnnotation: true,
		AcceptsIngressClassNameSpec:       false,
	},
	typeNeeded{
		PackageImportAlias:                "knativev1alpha1",
		PackageAlias:                      "Knativev1alpha1",
		Package:                           knativev1alpha1,
		Type:                              "Ingress",
		Plural:                            "ingresses",
		URL:                               "networking.internal.knative.dev",
		CacheType:                         "Ingresses",
		AcceptsIngressClassNameAnnotation: true,
		AcceptsIngressClassNameSpec:       false,
	},
}

func main() {
	if err := inputControllersNeeded.generate(); err != nil {
		fmt.Fprintf(os.Stderr, "could not generate input controllers: %v", err)
		os.Exit(1)
	}
}

// -----------------------------------------------------------------------------
// Private Functions - Helper
// -----------------------------------------------------------------------------

// header produces a skeleton of the controller file to be generated.
func header() (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)

	boilerPlate, err := ioutil.ReadFile("hack/boilerplate.go.txt")
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(boilerPlate)
	if err != nil {
		return nil, err
	}

	_, err = buf.WriteString(headerTemplate)
	return buf, err
}

// -----------------------------------------------------------------------------
// Generator
// -----------------------------------------------------------------------------

// typesNeeded is a list of Kubernetes API types which are supported
// by the Kong Kubernetes Ingress Controller and need to have "input"
// controllers generated for them.
type typesNeeded []typeNeeded

// generate generates a controller/input/<controller>.go Kubernetes controller
// for every supported type populated in the list.
func (types typesNeeded) generate() error {
	contents, err := header()
	if err != nil {
		return err
	}

	for _, t := range types {
		if err := t.generate(contents); err != nil {
			return err
		}
	}

	return ioutil.WriteFile(outputFile, contents.Bytes(), 0644)
}

type typeNeeded struct {
	PackageImportAlias string
	PackageAlias       string
	Package            string
	Type               string
	Plural             string
	URL                string
	CacheType          string

	// AcceptsIngressClassNameAnnotation indicates that the object accepts (and the controller will listen to)
	// the "kubernetes.io/ingress.class" annotation to decide whether or not the object is supported.
	AcceptsIngressClassNameAnnotation bool

	// AcceptsIngressClassNameSpec indicates the the object indicates the ingress.class that should support it via
	// an attribute in its specification named .IngressClassName
	AcceptsIngressClassNameSpec bool
}

func (t *typeNeeded) generate(contents *bytes.Buffer) error {
	tmpl, err := template.New("controller").Parse(controllerTemplate)
	if err != nil {
		return err
	}
	return tmpl.Execute(contents, t)
}

// -----------------------------------------------------------------------------
// Templates
// -----------------------------------------------------------------------------

var headerTemplate = `
// Code generated by Kong; DO NOT EDIT.

package configuration

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	netv1 "k8s.io/api/networking/v1"
	netv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kongv1 "github.com/kong/kubernetes-ingress-controller/railgun/apis/configuration/v1"
	kongv1alpha1 "github.com/kong/kubernetes-ingress-controller/railgun/apis/configuration/v1alpha1"
	kongv1beta1 "github.com/kong/kubernetes-ingress-controller/railgun/apis/configuration/v1beta1"

	"github.com/kong/kubernetes-ingress-controller/railgun/internal/ctrlutils"
	"github.com/kong/kubernetes-ingress-controller/railgun/internal/proxy"
)
`

var controllerTemplate = `
// -----------------------------------------------------------------------------
// {{.PackageAlias}} {{.Type}}
// -----------------------------------------------------------------------------

// {{.PackageAlias}}{{.Type}} reconciles a Ingress object
type {{.PackageAlias}}{{.Type}}Reconciler struct {
	client.Client

	Log    logr.Logger
	Scheme *runtime.Scheme
	Proxy  proxy.Proxy
{{- if or .AcceptsIngressClassNameSpec .AcceptsIngressClassNameAnnotation}}

	IngressClassName string
{{- end}}
}

// SetupWithManager sets up the controller with the Manager.
func (r *{{.PackageAlias}}{{.Type}}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
{{- if .AcceptsIngressClassNameAnnotation}}
	preds := ctrlutils.GeneratePredicateFuncsForIngressClassFilter(r.IngressClassName, {{.AcceptsIngressClassNameSpec}}, true)
	return ctrl.NewControllerManagedBy(mgr).For(&{{.PackageImportAlias}}.{{.Type}}{}, builder.WithPredicates(preds)).Complete(r)
{{- else}}
	return ctrl.NewControllerManagedBy(mgr).For(&{{.PackageImportAlias}}.{{.Type}}{}).Complete(r)
{{- end}}
}

//+kubebuilder:rbac:groups={{.URL}},resources={{.Plural}},verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups={{.URL}},resources={{.Plural}}/status,verbs=get;update;patch
//+kubebuilder:rbac:groups={{.URL}},resources={{.Plural}}/finalizers,verbs=update

// Reconcile processes the watched objects
func (r *{{.PackageAlias}}{{.Type}}Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("{{.PackageAlias}}{{.Type}}", req.NamespacedName)

	// get the relevant object
	obj := new({{.PackageImportAlias}}.{{.Type}})
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("reconciling resource", "namespace", req.Namespace, "name", req.Name)

	// clean the object up if it's being deleted
	if !obj.DeletionTimestamp.IsZero() && time.Now().After(obj.DeletionTimestamp.Time) {
		log.Info("resource is being deleted, its configuration will be removed", "type", "{{.Type}}", "namespace", req.Namespace, "name", req.Name)
		if err := r.Proxy.DeleteObject(obj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrlutils.CleanupFinalizer(ctx, r.Client, log, req.NamespacedName, obj)
	}
{{if .AcceptsIngressClassNameAnnotation}}
	// if the object is not configured with our ingress.class, then we need to ensure it's removed from the cache
	if !ctrlutils.MatchesIngressClassName(obj, r.IngressClassName) {
		log.Info("object missing ingress class, ensuring it's removed from configuration", req.Namespace, req.Name)
		if err := r.Proxy.DeleteObject(obj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
{{end}}
	// before we store cache data for this object, ensure that it has our finalizer set
	if !ctrlutils.HasFinalizer(obj, ctrlutils.KongIngressFinalizer) {
		log.Info("finalizer is not set for ingress object, setting it", req.Namespace, req.Name)
		finalizers := obj.GetFinalizers()
		obj.SetFinalizers(append(finalizers, ctrlutils.KongIngressFinalizer))
		if err := r.Client.Update(ctx, obj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// update the kong Admin API with the changes
	log.Info("updating the proxy with new {{.Type}}", "namespace", obj.Namespace, "name", obj.Name)
	if err := r.Proxy.UpdateObject(obj); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
`
