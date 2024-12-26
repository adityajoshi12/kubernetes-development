/*
Copyright 2024.

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

package controller

import (
	"bytes"
	"context"
	"fmt"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"html/template"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"

	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiv1 "github.com/adityajoshi12/devenv-operator/api/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const finalizerString = "finalizer.devenv.adityajoshi.online"
const resourceURL = "developerenv.adityajoshi.online"

// DeveloperEnvironmentReconciler reconciles a DeveloperEnvironment object
type DeveloperEnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=api.adityajoshi.online,resources=developerenvironments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.adityajoshi.online,resources=developerenvironments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=api.adityajoshi.online,resources=developerenvironments/finalizers,verbs=update
func (r *DeveloperEnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the DeveloperEnvironment instance
	devEnv := &apiv1.DeveloperEnvironment{}
	if err := r.Get(ctx, req.NamespacedName, devEnv); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if the object is being deleted

	if devEnv.DeletionTimestamp != nil {
		if containsString(devEnv.Finalizers, finalizerString) {
			// Run finalization logic for finalizer.devenv.adityajoshi.online
			if err := r.finalizeDeveloperEnvironment(ctx, devEnv); err != nil {
				return ctrl.Result{}, err
			}

			// Remove finalizer from the list and update it
			devEnv.Finalizers = removeString(devEnv.Finalizers, finalizerString)
			if err := r.Update(ctx, devEnv); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	// Add finalizer for this CR
	if !containsString(devEnv.Finalizers, finalizerString) {
		devEnv.Finalizers = append(devEnv.Finalizers, finalizerString)
		if err := r.Update(ctx, devEnv); err != nil {
			if apierrors.IsConflict(err) {
				// Re-fetch the latest version of the DeveloperEnvironment and retry
				if err := r.Get(ctx, req.NamespacedName, devEnv); err != nil {
					return ctrl.Result{}, err
				}
				devEnv.Finalizers = append(devEnv.Finalizers, finalizerString)
				if err := r.Update(ctx, devEnv); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				return ctrl.Result{}, err
			}
		}
	}
	// Update status
	if err := r.updateStatus(ctx, devEnv); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileDeveloperEnvironment(ctx, devEnv); err != nil {
		logger.Error(err, "Failed to reconcile developer environment")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	// Update status
	if err := r.updateStatus(ctx, devEnv); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
}

// Reconcile main logic
func (r *DeveloperEnvironmentReconciler) reconcileDeveloperEnvironment(
	ctx context.Context,
	devEnv *apiv1.DeveloperEnvironment,
) error {
	// 1. Create Namespace
	if err := r.ensureNamespace(ctx, devEnv); err != nil {
		return err
	}
	if err := r.setupCertificates(ctx, devEnv); err != nil {
		return err
	}

	// 3. Provision Development Tools
	if err := r.provisionDevelopmentTools(ctx, devEnv); err != nil {
		return err
	}

	// 4. Setup IDE (VS Code Server)
	if err := r.setupVSCodeServer(ctx, devEnv); err != nil {
		return err
	}

	// 5. Configure Dependencies
	if err := r.setupDatabase(ctx, devEnv); err != nil {
		return err
	}

	return nil
}

// Namespace creation
func (r *DeveloperEnvironmentReconciler) ensureNamespace(
	ctx context.Context,
	devEnv *apiv1.DeveloperEnvironment,
) error {
	ns := fmt.Sprintf("devenv-%s", devEnv.Name)

	// Check if the namespace exists
	namespace := &corev1.Namespace{}
	err := r.Get(ctx, client.ObjectKey{Name: ns}, namespace)
	if err != nil && apierrors.IsNotFound(err) {
		// Namespace does not exist, create it
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Labels: map[string]string{
					"managed-by":  "devenv-operator",
					"environment": devEnv.Name,
				},
			},
		}
		if err := r.Create(ctx, namespace); err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	} else if err != nil {
		// An error occurred while checking for the namespace
		return fmt.Errorf("failed to get namespace: %w", err)
	}
	return nil
}

func (r *DeveloperEnvironmentReconciler) provisionDevelopmentTools(
	ctx context.Context,
	devEnv *apiv1.DeveloperEnvironment,
) error {
	// Create a template for the installation script
	installScriptTemplate, err := template.New("install-tools").Parse(`#!/bin/bash
echo $SUDO_PASSWORD | sudo -S -v
export DEBIAN_FRONTEND=noninteractive
# Update package lists
sudo apt-get update

# Install core development tools
sudo apt-get install -y \
    git \
    curl \
    wget \
    build-essential \
    software-properties-common \
    ca-certificates \
    gnupg \
    lsb-release

# Install language-specific tools based on environment configuration
{{- if .Languages.Python }}
# Python tools
PYTHON_VERSION={{ if .Versions.Python }}{{ .Versions.Python }}{{ else }}3{{ end }}
sudo apt-get install -y python${PYTHON_VERSION} python${PYTHON_VERSION}-pip python${PYTHON_VERSION}-venv
pip${PYTHON_VERSION} install --upgrade pip
pip${PYTHON_VERSION} install poetry virtualenv
{{- end }}

{{- if .Languages.NodeJS }}
# Node.js and npm using nvm
NODEJS_VERSION={{ if .Versions.NodeJS }}{{ .Versions.NodeJS }}{{ else }}lts{{ end }}
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.4/install.sh | bash
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
nvm install ${NODEJS_VERSION}
nvm use ${NODEJS_VERSION}
npm install -g yarn pnpm
{{- end }}

{{- if .Languages.Go }}
# Go language
GO_VERSION={{ if .Versions.Go }}{{ .Versions.Go }}{{ else }}1.21.5{{ end }}
wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
sudo rm go${GO_VERSION}.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
{{- end }}

{{- if .Languages.Rust }}
# Rust toolchain
RUST_VERSION={{ if .Versions.Rust }}{{ .Versions.Rust }}{{ else }}stable{{ end }}
sudo curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain ${RUST_VERSION}
{{- end }}

# Install additional tools specified in the environment
{{- range .AdditionalTools }}
sudo apt-get install -y {{ . }}
{{- end }}

# Clean up
sudo apt-get clean
sudo rm -rf /var/lib/apt/lists/*

echo "Development tools installation complete!"
`)
	if err != nil {
		return fmt.Errorf("failed to parse installation script template: %w", err)
	}
	// Prepare the template data
	type TemplateData struct {
		Languages struct {
			Python bool
			NodeJS bool
			Go     bool
			Rust   bool
		}
		Versions struct {
			Python string
			NodeJS string
			Go     string
			Rust   string
		}
		AdditionalTools []string
	}

	// Populate template data
	templateData := TemplateData{
		Languages: struct {
			Python bool
			NodeJS bool
			Go     bool
			Rust   bool
		}{
			Python: devEnv.Spec.Language == "python",
			NodeJS: devEnv.Spec.Language == "nodejs",
			Go:     devEnv.Spec.Language == "go",
			Rust:   devEnv.Spec.Language == "rust",
		},
		Versions: struct {
			Python string
			NodeJS string
			Go     string
			Rust   string
		}{
			Python: devEnv.Spec.Version,
			NodeJS: devEnv.Spec.Version,
			Go:     devEnv.Spec.Version,
			Rust:   devEnv.Spec.Version,
		},
	}

	// Render the script
	var scriptContent bytes.Buffer
	if err := installScriptTemplate.Execute(&scriptContent, templateData); err != nil {
		return fmt.Errorf("failed to render installation script: %w", err)
	}

	// Create a ConfigMap to store the rendered installation script
	toolsConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-dev-tools-scripts", devEnv.Name),
			Namespace: devEnv.Namespace,
			Labels: map[string]string{
				"developer-env":     devEnv.Name,
				"developer-env-uid": string(devEnv.UID),
				"app":               "development-tools",
			},
		},
		Data: map[string]string{
			"install-tools.sh": scriptContent.String(),
		},
	}

	// Create or update the ConfigMap
	existingConfigMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      toolsConfigMap.Name,
		Namespace: toolsConfigMap.Namespace,
	}, existingConfigMap)

	if err != nil {
		if errors.IsNotFound(err) {
			// Create the ConfigMap if it doesn't exist
			if createErr := r.Create(ctx, toolsConfigMap); createErr != nil {
				return fmt.Errorf("failed to create tools ConfigMap: %w", createErr)
			}
		} else {
			return fmt.Errorf("failed to check existing ConfigMap: %w", err)
		}
	} else {
		// Update existing ConfigMap
		existingConfigMap.Data = toolsConfigMap.Data
		existingConfigMap.Labels = toolsConfigMap.Labels
		if updateErr := r.Update(ctx, existingConfigMap); updateErr != nil {
			return fmt.Errorf("failed to update tools ConfigMap: %w", updateErr)
		}
	}

	return nil
}

func (r *DeveloperEnvironmentReconciler) setupVSCodeServer(
	ctx context.Context,
	devEnv *apiv1.DeveloperEnvironment,
) error {
	// Generate a unique name for the VS Code server resources
	vsCodeServerName := fmt.Sprintf("%s-vscode-server", devEnv.Name)

	// Create a PersistentVolumeClaim for workspace persistence
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vscode-workspace", devEnv.Name),
			Namespace: devEnv.Namespace,
			Labels: map[string]string{
				"app":           "vscode-server",
				"developer-env": devEnv.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}

	// Create or update the PVC
	if err := r.Create(ctx, pvc); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create VS Code workspace PVC: %w", err)
		}
	}

	installExtensionCommand := &corev1.Lifecycle{}
	if len(devEnv.Spec.IDE.Extensions) > 0 {
		installExtensionCommand = &corev1.Lifecycle{
			PostStart: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/bin/bash",
						"-c",
						fmt.Sprintf("./config/tools/install-tools.sh && ./app/code-server/bin/code-server --extensions-dir /config/extensions --install-extension %s",
							strings.Join(devEnv.Spec.IDE.Extensions, " --install-extension ")),
					},
				},
			},
		}
	}

	// Create a deployment for the VS Code server
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vsCodeServerName,
			Namespace: devEnv.Namespace,
			Labels: map[string]string{
				"app":               "vscode-server",
				"developer-env":     devEnv.Name,
				"developer-env-uid": string(devEnv.UID),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: Ptr(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":           "vscode-server",
					"developer-env": devEnv.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":           "vscode-server",
						"developer-env": devEnv.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "vscode-server",
							Image:           "linuxserver/code-server:4.95.3",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8443,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "PUID",
									Value: "1000",
								},
								{
									Name:  "PGID",
									Value: "1000",
								},
								{
									Name: "PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: fmt.Sprintf("%s-vscode-password", devEnv.Name),
											},
											Key: "password",
										},
									},
								},
								{
									Name: "SUDO_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: fmt.Sprintf("%s-vscode-password", devEnv.Name),
											},
											Key: "password",
										},
									},
								},
							},

							Lifecycle: installExtensionCommand,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workspace",
									MountPath: "/config/workspace",
								},
								{
									Name:      "tools-script",
									MountPath: "/config/tools",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "workspace",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-vscode-workspace", devEnv.Name),
								},
							},
						},
						{
							Name: "tools-script",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-dev-tools-scripts", devEnv.Name),
									},
									DefaultMode: Ptr(int32(0777)),
								},
							},
						},
					},
				},
			},
		},
	}

	// Create or update the deployment
	if err := r.Create(ctx, deployment); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create VS Code server deployment: %w", err)
		}

		// If already exists, update the deployment
		existingDeployment := &appsv1.Deployment{}
		if getErr := r.Get(ctx, types.NamespacedName{
			Name:      vsCodeServerName,
			Namespace: devEnv.Namespace,
		}, existingDeployment); getErr != nil {
			return fmt.Errorf("failed to get existing VS Code server deployment: %w", getErr)
		}

		existingDeployment.Spec = deployment.Spec
		if updateErr := r.Update(ctx, existingDeployment); updateErr != nil {
			return fmt.Errorf("failed to update VS Code server deployment: %w", updateErr)
		}
	}

	// Create a service to expose the VS Code server
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vsCodeServerName,
			Namespace: devEnv.Namespace,
			Labels: map[string]string{
				"app":           "vscode-server",
				"developer-env": devEnv.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":           "vscode-server",
				"developer-env": devEnv.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8443,
					TargetPort: intstr.FromInt(8443),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	// Create or update the service
	if err := r.Create(ctx, service); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create VS Code server service: %w", err)
		}

		// If already exists, update the service
		existingService := &corev1.Service{}
		if getErr := r.Get(ctx, types.NamespacedName{
			Name:      vsCodeServerName,
			Namespace: devEnv.Namespace,
		}, existingService); getErr != nil {
			return fmt.Errorf("failed to get existing VS Code server service: %w", getErr)
		}

		existingService.Spec = service.Spec
		if updateErr := r.Update(ctx, existingService); updateErr != nil {
			return fmt.Errorf("failed to update VS Code server service: %w", updateErr)
		}
	}

	secretName := fmt.Sprintf("%s-vscode-password", devEnv.Name)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: devEnv.Namespace,
			Labels: map[string]string{
				"app":           "vscode-server",
				"developer-env": devEnv.Name,
			},
		},
		StringData: map[string]string{
			"password": devEnv.Spec.IDE.PasswordSecret,
		},
	}

	if err := r.Create(ctx, secret); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create VS Code server secret: %w", err)
		}
	}
	ingressClass := "nginx"
	ingressName := fmt.Sprintf("%s-vscode-ingress", devEnv.Name)
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: devEnv.Namespace,
			Labels: map[string]string{
				"app":           "vscode-server",
				"developer-env": devEnv.Name,
			},
			Annotations: map[string]string{
				"cert-manager.io/issuer":                         "selfsigned-cluster-issuer",
				"kubernetes.io/ingress.class":                    ingressClass,
				"nginx.ingress.kubernetes.io/force-ssl-redirect": "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClass,
			Rules: []networkingv1.IngressRule{
				{
					Host: fmt.Sprintf("%s.%s", devEnv.Name, resourceURL),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: Ptr(networkingv1.PathTypePrefix),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: fmt.Sprintf("%s-vscode-server", devEnv.Name),
											Port: networkingv1.ServiceBackendPort{
												Number: 8443,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: []networkingv1.IngressTLS{
				{
					Hosts: []string{
						fmt.Sprintf("%s.%s", devEnv.Name, resourceURL),
					},
					SecretName: fmt.Sprintf("%s.%s", devEnv.Name, resourceURL),
				},
			},
		},
	}

	if err := r.Create(ctx, ingress); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create VS Code server ingress: %w", err)
		}

		// If already exists, update the ingress
		existingIngress := &networkingv1.Ingress{}
		if getErr := r.Get(ctx, types.NamespacedName{
			Name:      ingressName,
			Namespace: devEnv.Namespace,
		}, existingIngress); getErr != nil {
			return fmt.Errorf("failed to get existing VS Code server ingress: %w", getErr)
		}

		existingIngress.Spec = ingress.Spec
		if updateErr := r.Update(ctx, existingIngress); updateErr != nil {
			return fmt.Errorf("failed to update VS Code server ingress: %w", updateErr)
		}
	}

	return nil
}

func (r *DeveloperEnvironmentReconciler) setupDatabase(ctx context.Context, devEnv *apiv1.DeveloperEnvironment) error {
	dbType := devEnv.Spec.Database.Type
	dbVersion := devEnv.Spec.Database.Version
	dbName := fmt.Sprintf("%s-database", devEnv.Name)

	var containerPorts []corev1.ContainerPort
	var envVars []corev1.EnvVar
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume

	switch dbType {
	case "postgres":
		containerPorts = []corev1.ContainerPort{
			{
				Name:          "db",
				ContainerPort: 5432,
			},
		}
		envVars = []corev1.EnvVar{
			{
				Name:  "POSTGRES_DB",
				Value: "postgres",
			},
			{
				Name:  "POSTGRES_USER",
				Value: "postgres",
			},
			{
				Name:  "POSTGRES_PASSWORD",
				Value: "postgres",
			},
			{
				Name:  "PGDATA",
				Value: "/var/lib/postgresql/data/pgdata",
			},
		}
		volumeMounts = []corev1.VolumeMount{
			{
				Name:      "db-data",
				MountPath: "/var/lib/postgresql/data",
			},
		}
		volumes = []corev1.Volume{
			{
				Name: "db-data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("%s-db-pvc", devEnv.Name),
					},
				},
			},
		}
	case "redis":
		containerPorts = []corev1.ContainerPort{
			{
				Name:          "db",
				ContainerPort: 6379,
			},
		}
		envVars = []corev1.EnvVar{
			{
				Name:  "REDIS_PASSWORD",
				Value: "password",
			},
		}
		volumeMounts = []corev1.VolumeMount{
			{
				Name:      "db-data",
				MountPath: "/data",
			},
		}
		volumes = []corev1.Volume{
			{
				Name: "db-data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: fmt.Sprintf("%s-db-pvc", devEnv.Name),
					},
				},
			},
		}
	}

	// Define the database PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-db-pvc", devEnv.Name),
			Namespace: devEnv.Namespace,
			Labels: map[string]string{
				"app":           "database",
				"developer-env": devEnv.Name,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	}

	// Check if the PVC already exists
	existingPVC := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{
		Name:      pvc.Name,
		Namespace: pvc.Namespace,
	}, existingPVC)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// PVC does not exist, create it
			if createErr := r.Create(ctx, pvc); createErr != nil {
				return fmt.Errorf("failed to create database PVC: %w", createErr)
			}
		} else {
			return fmt.Errorf("failed to check existing PVC: %w", err)
		}
	}

	// Define the database deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbName,
			Namespace: devEnv.Namespace,
			Labels: map[string]string{
				"app":           "database",
				"developer-env": devEnv.Name,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: Ptr(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":           "database",
					"developer-env": devEnv.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":           "database",
						"developer-env": devEnv.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:         "database",
							Image:        fmt.Sprintf("%s:%s", dbType, dbVersion),
							Ports:        containerPorts,
							Env:          envVars,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	// Create or update the deployment
	if err := r.Create(ctx, deployment); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create database deployment: %w", err)
		}

		// If already exists, update the deployment
		existingDeployment := &appsv1.Deployment{}
		if getErr := r.Get(ctx, types.NamespacedName{
			Name:      dbName,
			Namespace: devEnv.Namespace,
		}, existingDeployment); getErr != nil {
			return fmt.Errorf("failed to get existing database deployment: %w", getErr)
		}

		existingDeployment.Spec = deployment.Spec
		if updateErr := r.Update(ctx, existingDeployment); updateErr != nil {
			return fmt.Errorf("failed to update database deployment: %w", updateErr)
		}
	}

	// Define the database service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbName,
			Namespace: devEnv.Namespace,
			Labels: map[string]string{
				"app":           "database",
				"developer-env": devEnv.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":           "database",
				"developer-env": devEnv.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "db",
					Port:       containerPorts[0].ContainerPort,
					TargetPort: intstr.FromInt32(containerPorts[0].ContainerPort),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	// Create or update the service
	if err := r.Create(ctx, service); err != nil {
		if !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create database service: %w", err)
		}

		// If already exists, update the service
		existingService := &corev1.Service{}
		if getErr := r.Get(ctx, types.NamespacedName{
			Name:      dbName,
			Namespace: devEnv.Namespace,
		}, existingService); getErr != nil {
			return fmt.Errorf("failed to get existing database service: %w", getErr)
		}

		existingService.Spec = service.Spec
		if updateErr := r.Update(ctx, existingService); updateErr != nil {
			return fmt.Errorf("failed to update database service: %w", updateErr)
		}
	}

	return nil
}

// Update status of the DevEnv resource
func (r *DeveloperEnvironmentReconciler) updateStatus(
	ctx context.Context,
	devEnv *apiv1.DeveloperEnvironment,
) error {
	devEnv.Status.Phase = "Ready"
	devEnv.Status.LastUpdated = metav1.Now()

	return r.Status().Update(ctx, devEnv)
}

func (r *DeveloperEnvironmentReconciler) finalizeDeveloperEnvironment(ctx context.Context, devEnv *apiv1.DeveloperEnvironment) error {
	// Delete Namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("devenv-%s", devEnv.Name),
		},
	}
	if err := r.Delete(ctx, namespace); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	// Delete Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vscode-server", devEnv.Name),
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, deployment); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	// Delete Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vscode-server", devEnv.Name),
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, service); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	// Delete Secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vscode-password", devEnv.Name),
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	// Delete Ingress
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vscode-ingress", devEnv.Name),
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, ingress); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete ingress: %w", err)
	}

	// Delete Tools configmap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-dev-tools-scripts", devEnv.Name),
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, configMap); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete tools configmap: %w", err)
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-vscode-workspace", devEnv.Name),
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, pvc); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete pvc: %w", err)
	}
	// Delete Issuer
	issuer := &certmanagerv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selfsigned-cluster-issuer",
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, issuer); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete issuer: %w", err)
	}

	// Delete Certificate
	certificate := &certmanagerv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s.%s", devEnv.Name, resourceURL),
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, certificate); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete certificate: %w", err)
	}

	// Delete Database Deployment
	dbDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-database", devEnv.Name),
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, dbDeployment); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete database deployment: %w", err)
	}

	// Delete Database Service
	dbService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-database", devEnv.Name),
			Namespace: devEnv.Namespace,
		},
	}
	if err := r.Delete(ctx, dbService); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete database service: %w", err)
	}

	return nil
}
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeveloperEnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := certmanagerv1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.DeveloperEnvironment{}).
		Complete(r)
}

func (r *DeveloperEnvironmentReconciler) setupCertificates(ctx context.Context, devEnv *apiv1.DeveloperEnvironment) error {
	// Check if the Issuer exists
	selfSignedClusterIssuerName := "selfsigned-cluster-issuer"
	existingIssuer := &certmanagerv1.Issuer{}
	err := r.Get(ctx, client.ObjectKey{Name: selfSignedClusterIssuerName, Namespace: devEnv.Namespace}, existingIssuer)
	if err != nil && apierrors.IsNotFound(err) {
		// Issuer does not exist, create it
		issuer := &certmanagerv1.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      selfSignedClusterIssuerName,
				Namespace: devEnv.Namespace,
			},
			Spec: certmanagerv1.IssuerSpec{
				IssuerConfig: certmanagerv1.IssuerConfig{
					SelfSigned: &certmanagerv1.SelfSignedIssuer{},
				},
			},
		}
		if err := r.Create(ctx, issuer); err != nil {
			return fmt.Errorf("failed to create Issuer: %w", err)
		}
	} else if err != nil {
		// An error occurred while checking for the Issuer
		return fmt.Errorf("failed to get Issuer: %w", err)
	}

	// Check if the Certificate exists
	existingCertificate := &certmanagerv1.Certificate{}
	certificateName := fmt.Sprintf("%s.%s", devEnv.Name, resourceURL)
	err = r.Get(ctx, client.ObjectKey{Name: certificateName, Namespace: devEnv.Namespace}, existingCertificate)
	if err != nil && apierrors.IsNotFound(err) {
		// Certificate does not exist, create it
		certificate := &certmanagerv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      certificateName,
				Namespace: devEnv.Namespace,
			},
			Spec: certmanagerv1.CertificateSpec{
				IsCA:       true,
				CommonName: certificateName,
				SecretName: certificateName,
				PrivateKey: &certmanagerv1.CertificatePrivateKey{
					Algorithm: certmanagerv1.ECDSAKeyAlgorithm,
					Size:      256,
				},
				IssuerRef: cmmeta.ObjectReference{
					Name:  selfSignedClusterIssuerName,
					Kind:  certmanagerv1.IssuerKind,
					Group: certmanagerv1.SchemeGroupVersion.Group,
				},
			},
		}
		if err := r.Create(ctx, certificate); err != nil {
			return fmt.Errorf("failed to create Certificate: %w", err)
		}
	} else if err != nil {
		// An error occurred while checking for the Certificate
		return fmt.Errorf("failed to get Certificate: %w", err)
	}
	return nil
}
func Ptr[T any](v T) *T {
	return &v
}
