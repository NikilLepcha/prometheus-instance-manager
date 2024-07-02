package controller

import (
	"context"
	"fmt"

	// "fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	instancemanagerv1alpha1 "github.com/NikilLepcha/prometheus-instance-manager/api/v1alpha1"
)

// InstanceManagerReconciler reconciles a InstanceManager object
type InstanceManagerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=instancemanager.example.org,resources=instancemanagers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=instancemanager.example.org,resources=instancemanagers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=instancemanager.example.org,resources=instancemanagers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the InstanceManager object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *InstanceManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Enter Reconcile", "req", req)

	promDep := &instancemanagerv1alpha1.InstanceManager{}
	err := r.Get(ctx, req.NamespacedName, promDep)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Prometheus Custom Resource", "Dep", promDep)

	// Define a new Deployment object
	newDeployment := constructDeployment(promDep)

	log.Info("Checking if the Deployment already exist")

	// Check if the Deployment already exists
	found := &appsv1.Deployment{}
	err = r.Get(ctx, client.ObjectKey{Name: newDeployment.Name, Namespace: newDeployment.Namespace}, found)
	if err != nil && client.IgnoreNotFound(err) == nil {
		log.Info("Deployment not found, Creating new deployment")
		err = r.Create(ctx, newDeployment)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else if err == nil {
		log.Info("Existing Deployment Found")
		if !reflect.DeepEqual(newDeployment.Spec, found.Spec) {
			log.Info("Existing state is different, Updating it to desired state")
			found.Spec = newDeployment.Spec
			err = r.Update(ctx, found)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		// Update status
		log.Info("Updating the Status")
		promDep.Status.AvailableReplicas = *found.Spec.Replicas
		err = r.Status().Update(ctx, promDep)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	log.Info("Setting promDep instance as the owner and controller of the deployment")
	// Set the Prometheus instance as the owner and controller
	if err = controllerutil.SetControllerReference(promDep, newDeployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Define a new ConfigMap object
	configData := fmt.Sprintf(`
global:
  scrape_interval: %s
scrape_configs:
  - job_name: 'prometheus'
    static_configs:
    - targets: ['localhost:9090']
`, promDep.Spec.ScrapeInterval)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      promDep.Name + "-config",
			Namespace: promDep.Namespace,
		},
		Data: map[string]string{
			"prometheus.yml": configData,
		},
	}

	log.Info("Checking if the ConfigMap already exist")
	// Check if the ConfigMap already exists
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Get(ctx, client.ObjectKey{Name: configMap.Name, Namespace: configMap.Namespace}, foundConfigMap)
	if err != nil && client.IgnoreNotFound(err) == nil {
		log.Info("ConfigMap not found, Creating new ConfigMap")
		// Create the ConfigMap
		err = r.Create(ctx, configMap)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else if err == nil {
		log.Info("Existing ConfigMap Found")
		// Update the existing ConfigMap if needed
		if !reflect.DeepEqual(configMap.Data, foundConfigMap.Data) {
			log.Info("Existing config of ConfigMap is different, Updating it to desired config")
			foundConfigMap.Data = configMap.Data
			err = r.Update(ctx, foundConfigMap)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	log.Info("Setting promDep instance as the owner and controller of the ConfigMap")
	// Set the Prometheus instance as the owner and controller
	if err = controllerutil.SetControllerReference(promDep, configMap, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstanceManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&instancemanagerv1alpha1.InstanceManager{}).
		Complete(r)
}

func constructDeployment(p *instancemanagerv1alpha1.InstanceManager) *appsv1.Deployment {
	labels := labelForPrometheusCrd(p)
	replicas := p.Spec.Replicas

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.Name,
			Namespace: p.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "prometheus",
						Image: p.Spec.Image,
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							ContainerPort: 9090,
						}},
						Args: []string{
							"--config.file=/etc/prometheus/prometheus.yml",
						},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config-volume",
							MountPath: "/etc/prometheus",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: p.Name + "-config",
								},
							},
						},
					}},
				},
			},
		},
	}

	return deploy
}

func labelForPrometheusCrd(p *instancemanagerv1alpha1.InstanceManager) map[string]string {
	return map[string]string{
		"app":       p.Name,
		"createdBy": "prometheusInstanceManager_crd",
	}
}
