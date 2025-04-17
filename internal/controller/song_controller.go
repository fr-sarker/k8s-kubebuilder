/*
Copyright 2025.

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
	"context"
	musicv1 "github.com/frsarker/k8s-kubebuilder/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// SongReconciler reconciles a Song object
type SongReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=music.frsarker.kubebuilder.io,resources=songs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=music.frsarker.kubebuilder.io,resources=songs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=music.frsarker.kubebuilder.io,resources=songs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Song object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *SongReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var mysong musicv1.Song
	if err := r.Get(ctx, req.NamespacedName, &mysong); err != nil {
		log.Error(err, "unable to fetch song")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create or update deployment
	var deploy appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &deploy); err != nil {
		log.Info("creating deployment", "key", req.NamespacedName)
		if err := r.Create(ctx, newDeployment(mysong)); err != nil {
			log.Error(err, "unable to create deployment", "key", req.NamespacedName)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		} else {
			log.Info("created deployment", "key", req.NamespacedName)
			if deploy.Status.AvailableReplicas != mysong.Spec.Replicas {
				deploy.Spec.Replicas = &mysong.Spec.Replicas
				if err := r.Update(ctx, &deploy); err != nil {
					log.Error(err, "unable to update deployment", "key", req.NamespacedName)
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}
			}
		}

		mysong.Status.AvailableReplicas = deploy.Status.AvailableReplicas
		_ = r.Status().Update(ctx, &mysong)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SongReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&musicv1.Song{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func newDeployment(song musicv1.Song) client.Object {
	labels := map[string]string{
		"app":    song.Name,
		"artist": song.Spec.Artist,
		"title":  song.Spec.Title,
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      song.Name,
			Namespace: song.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(&song, musicv1.GroupVersion.WithKind("Song")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &song.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "song-container",
							Image: song.Spec.Title, // assuming title is the image name
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}
}
