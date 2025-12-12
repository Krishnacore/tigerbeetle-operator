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
	"fmt"
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	databasev1alpha1 "github.com/krishnacore/tigerbeetle-operator/api/v1alpha1"
)

const (
	typeAvailable   = "Available"
	typeProgressing = "Progressing"
	typeDegraded    = "Degraded"

	finalizerName = "database.tigerbeetle.com/finalizer"

	defaultImage    = "ghcr.io/krishnacore/tigerbeetle"
	defaultTag      = "latest"
	defaultReplicas = 3
	defaultPort     = 3000
	tigerBeetlePort = 3000
	dataPath        = "/var/lib/tigerbeetle"
)

// TigerBeetleReconciler reconciles a TigerBeetle object
type TigerBeetleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=database.tigerbeetle.com,resources=tigerbeetles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.tigerbeetle.com,resources=tigerbeetles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=database.tigerbeetle.com,resources=tigerbeetles/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *TigerBeetleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	tb := &databasev1alpha1.TigerBeetle{}
	if err := r.Get(ctx, req.NamespacedName, tb); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !tb.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(tb, finalizerName) {
			log.Info("Performing finalizer operations")
			controllerutil.RemoveFinalizer(tb, finalizerName)
			if err := r.Update(ctx, tb); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(tb, finalizerName) {
		controllerutil.AddFinalizer(tb, finalizerName)
		if err := r.Update(ctx, tb); err != nil {
			return ctrl.Result{}, err
		}
	}

	r.setDefaults(tb)

	meta.SetStatusCondition(&tb.Status.Conditions, metav1.Condition{
		Type:               typeProgressing,
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciling",
		Message:            "Reconciling TigerBeetle resources",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.reconcileServiceAccount(ctx, tb); err != nil {
		log.Error(err, "Failed to reconcile ServiceAccount")
		return r.setDegradedCondition(ctx, tb, err)
	}

	if err := r.reconcileService(ctx, tb); err != nil {
		log.Error(err, "Failed to reconcile Service")
		return r.setDegradedCondition(ctx, tb, err)
	}

	if err := r.reconcileStatefulSet(ctx, tb); err != nil {
		log.Error(err, "Failed to reconcile StatefulSet")
		return r.setDegradedCondition(ctx, tb, err)
	}

	if err := r.updateStatus(ctx, tb); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *TigerBeetleReconciler) setDefaults(tb *databasev1alpha1.TigerBeetle) {
	if tb.Spec.Replicas == 0 {
		tb.Spec.Replicas = defaultReplicas
	}
	if tb.Spec.Image.Repository == "" {
		tb.Spec.Image.Repository = defaultImage
	}
	if tb.Spec.Image.Tag == "" {
		tb.Spec.Image.Tag = defaultTag
	}
	if tb.Spec.Image.PullPolicy == "" {
		tb.Spec.Image.PullPolicy = corev1.PullIfNotPresent
	}
	if tb.Spec.Storage.Size.IsZero() {
		tb.Spec.Storage.Size = resource.MustParse("10Gi")
	}
	if tb.Spec.Service.Type == "" {
		tb.Spec.Service.Type = corev1.ServiceTypeClusterIP
	}
	if tb.Spec.Service.Port == 0 {
		tb.Spec.Service.Port = defaultPort
	}
}

func (r *TigerBeetleReconciler) setDegradedCondition(ctx context.Context, tb *databasev1alpha1.TigerBeetle, err error) (ctrl.Result, error) {
	meta.SetStatusCondition(&tb.Status.Conditions, metav1.Condition{
		Type:               typeDegraded,
		Status:             metav1.ConditionTrue,
		Reason:             "ReconcileError",
		Message:            err.Error(),
		LastTransitionTime: metav1.Now(),
	})
	tb.Status.Phase = "Degraded"
	if statusErr := r.Status().Update(ctx, tb); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: 10 * time.Second}, err
}

func (r *TigerBeetleReconciler) reconcileServiceAccount(ctx context.Context, tb *databasev1alpha1.TigerBeetle) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tb.Name,
			Namespace: tb.Namespace,
		},
	}

	if err := controllerutil.SetControllerReference(tb, sa, r.Scheme); err != nil {
		return err
	}

	found := &corev1.ServiceAccount{}
	err := r.Get(ctx, types.NamespacedName{Name: sa.Name, Namespace: sa.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, sa)
	}
	return err
}

func (r *TigerBeetleReconciler) reconcileService(ctx context.Context, tb *databasev1alpha1.TigerBeetle) error {
	headlessSvc := r.buildHeadlessService(tb)
	if err := controllerutil.SetControllerReference(tb, headlessSvc, r.Scheme); err != nil {
		return err
	}

	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: headlessSvc.Name, Namespace: headlessSvc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, headlessSvc)
	} else if err != nil {
		return err
	}

	found.Spec.Ports = headlessSvc.Spec.Ports
	return r.Update(ctx, found)
}

func (r *TigerBeetleReconciler) buildHeadlessService(tb *databasev1alpha1.TigerBeetle) *corev1.Service {
	labels := r.labelsForTigerBeetle(tb)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        tb.Name,
			Namespace:   tb.Namespace,
			Labels:      labels,
			Annotations: tb.Spec.Service.Annotations,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "tigerbeetle",
					Port:       tigerBeetlePort,
					TargetPort: intstr.FromInt(tigerBeetlePort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			PublishNotReadyAddresses: true,
		},
	}
}

func (r *TigerBeetleReconciler) reconcileStatefulSet(ctx context.Context, tb *databasev1alpha1.TigerBeetle) error {
	sts := r.buildStatefulSet(tb)
	if err := controllerutil.SetControllerReference(tb, sts, r.Scheme); err != nil {
		return err
	}

	found := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		return r.Create(ctx, sts)
	} else if err != nil {
		return err
	}

	found.Spec.Replicas = sts.Spec.Replicas
	found.Spec.Template = sts.Spec.Template
	return r.Update(ctx, found)
}

func (r *TigerBeetleReconciler) buildStatefulSet(tb *databasev1alpha1.TigerBeetle) *appsv1.StatefulSet {
	labels := r.labelsForTigerBeetle(tb)
	replicas := tb.Spec.Replicas

	podLabels := make(map[string]string)
	for k, v := range labels {
		podLabels[k] = v
	}
	for k, v := range tb.Spec.PodLabels {
		podLabels[k] = v
	}

	initCommand := r.buildInitCommand(tb)
	startCommand := r.buildStartCommand(tb)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tb.Name,
			Namespace: tb.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			ServiceName:         tb.Name,
			PodManagementPolicy: appsv1.ParallelPodManagement,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: tb.Spec.PodAnnotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: r.getServiceAccountName(tb),
					SecurityContext:    tb.Spec.SecurityContext,
					ImagePullSecrets:   tb.Spec.ImagePullSecrets,
					NodeSelector:       tb.Spec.NodeSelector,
					Tolerations:        tb.Spec.Tolerations,
					PriorityClassName:  tb.Spec.PriorityClassName,
					RestartPolicy:      corev1.RestartPolicyAlways,
					InitContainers: []corev1.Container{
						{
							Name:            "init-tigerbeetle",
							Image:           fmt.Sprintf("%s:%s", tb.Spec.Image.Repository, tb.Spec.Image.Tag),
							ImagePullPolicy: tb.Spec.Image.PullPolicy,
							Command:         []string{"/bin/sh", "-c", initCommand},
							Env: []corev1.EnvVar{
								{Name: "CLUSTER", Value: strconv.Itoa(tb.Spec.ClusterID)},
								{Name: "REPLICA_COUNT", Value: strconv.Itoa(int(tb.Spec.Replicas))},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: dataPath},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "tigerbeetle",
							Image:           fmt.Sprintf("%s:%s", tb.Spec.Image.Repository, tb.Spec.Image.Tag),
							ImagePullPolicy: tb.Spec.Image.PullPolicy,
							Command:         []string{"/bin/sh", "-c", startCommand},
							Env: append([]corev1.EnvVar{
								{Name: "CLUSTER", Value: strconv.Itoa(tb.Spec.ClusterID)},
							}, tb.Spec.Env...),
							Ports: []corev1.ContainerPort{
								{
									Name:          "tigerbeetle",
									ContainerPort: tigerBeetlePort,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Resources: tb.Spec.Resources,
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: dataPath},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "data",
						Labels:      labels,
						Annotations: tb.Spec.Storage.Annotations,
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: tb.Spec.Storage.StorageClassName,
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: tb.Spec.Storage.Size,
							},
						},
					},
				},
			},
		},
	}

	r.applyAffinity(sts, tb)
	r.applyTopologySpreadConstraints(sts, tb)

	return sts
}

func (r *TigerBeetleReconciler) buildInitCommand(tb *databasev1alpha1.TigerBeetle) string {
	return `set -ex
REPLICA=${HOSTNAME##*-}
FMT_CLUSTER=$(printf "%010d" $CLUSTER)
FMT_REPLICA=$(printf "%03d" $REPLICA)
DATA_FILE=/var/lib/tigerbeetle/cluster_${FMT_CLUSTER}_replica_${FMT_REPLICA}.tigerbeetle
if [ ! -f "$DATA_FILE" ]; then
  ./tigerbeetle format --cluster=$CLUSTER --replica=$REPLICA --replica-count=$REPLICA_COUNT $DATA_FILE
fi`
}

func (r *TigerBeetleReconciler) buildStartCommand(tb *databasev1alpha1.TigerBeetle) string {
	return fmt.Sprintf(`set -ex
REPLICA=${HOSTNAME##*-}
FMT_CLUSTER=$(printf "%%010d" $CLUSTER)
FMT_REPLICA=$(printf "%%03d" $REPLICA)
ADDRESSES=""
for n in $(seq 0 %d); do
  if [ "$n" != "$REPLICA" ]; then
    ADDRESSES="$ADDRESSES,%s-${n}.%s.%s.svc.cluster.local:%d"
  else
    ADDRESSES="$ADDRESSES,0.0.0.0:%d"
  fi
done
ADDRESSES=${ADDRESSES#,}
DATA_FILE=/var/lib/tigerbeetle/cluster_${FMT_CLUSTER}_replica_${FMT_REPLICA}.tigerbeetle
sleep 5
./tigerbeetle start --addresses=$ADDRESSES $DATA_FILE`,
		tb.Spec.Replicas-1, tb.Name, tb.Name, tb.Namespace, tigerBeetlePort, tigerBeetlePort)
}

func (r *TigerBeetleReconciler) applyAffinity(sts *appsv1.StatefulSet, tb *databasev1alpha1.TigerBeetle) {
	if tb.Spec.Affinity != nil {
		sts.Spec.Template.Spec.Affinity = tb.Spec.Affinity
		return
	}

	if tb.Spec.PodAntiAffinity != nil && tb.Spec.PodAntiAffinity.Type != "" {
		labels := r.labelsForTigerBeetle(tb)
		labelSelector := &metav1.LabelSelector{MatchLabels: labels}

		affinity := &corev1.Affinity{}
		topologyKey := tb.Spec.PodAntiAffinity.TopologyKey
		if topologyKey == "" {
			topologyKey = "kubernetes.io/hostname"
		}

		switch tb.Spec.PodAntiAffinity.Type {
		case "hard":
			affinity.PodAntiAffinity = &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						TopologyKey:   topologyKey,
						LabelSelector: labelSelector,
					},
				},
			}
		case "soft":
			weight := tb.Spec.PodAntiAffinity.Weight
			if weight == 0 {
				weight = 100
			}
			affinity.PodAntiAffinity = &corev1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
					{
						Weight: weight,
						PodAffinityTerm: corev1.PodAffinityTerm{
							TopologyKey:   topologyKey,
							LabelSelector: labelSelector,
						},
					},
				},
			}
		}
		sts.Spec.Template.Spec.Affinity = affinity
	}
}

func (r *TigerBeetleReconciler) applyTopologySpreadConstraints(sts *appsv1.StatefulSet, tb *databasev1alpha1.TigerBeetle) {
	if tb.Spec.TopologySpreadConstraints == nil {
		return
	}

	tsc := tb.Spec.TopologySpreadConstraints
	labels := r.labelsForTigerBeetle(tb)

	maxSkew := tsc.MaxSkew
	if maxSkew == 0 {
		maxSkew = 1
	}

	topologyKey := tsc.TopologyKey
	if topologyKey == "" {
		topologyKey = "topology.kubernetes.io/zone"
	}

	whenUnsatisfiable := tsc.WhenUnsatisfiable
	if whenUnsatisfiable == "" {
		whenUnsatisfiable = corev1.ScheduleAnyway
	}

	sts.Spec.Template.Spec.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           maxSkew,
			TopologyKey:       topologyKey,
			WhenUnsatisfiable: whenUnsatisfiable,
			LabelSelector:     &metav1.LabelSelector{MatchLabels: labels},
		},
	}
}

func (r *TigerBeetleReconciler) getServiceAccountName(tb *databasev1alpha1.TigerBeetle) string {
	if tb.Spec.ServiceAccountName != "" {
		return tb.Spec.ServiceAccountName
	}
	return tb.Name
}

func (r *TigerBeetleReconciler) labelsForTigerBeetle(tb *databasev1alpha1.TigerBeetle) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "tigerbeetle",
		"app.kubernetes.io/instance":   tb.Name,
		"app.kubernetes.io/managed-by": "tigerbeetle-operator",
		"app.kubernetes.io/component":  "database",
	}
}

func (r *TigerBeetleReconciler) updateStatus(ctx context.Context, tb *databasev1alpha1.TigerBeetle) error {
	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: tb.Name, Namespace: tb.Namespace}, sts); err != nil {
		if errors.IsNotFound(err) {
			tb.Status.ReadyReplicas = 0
			tb.Status.Phase = "Pending"
		} else {
			return err
		}
	} else {
		tb.Status.ReadyReplicas = sts.Status.ReadyReplicas

		if sts.Status.ReadyReplicas == *sts.Spec.Replicas {
			tb.Status.Phase = "Running"
			meta.SetStatusCondition(&tb.Status.Conditions, metav1.Condition{
				Type:               typeAvailable,
				Status:             metav1.ConditionTrue,
				Reason:             "AllReplicasReady",
				Message:            fmt.Sprintf("All %d replicas are ready", sts.Status.ReadyReplicas),
				LastTransitionTime: metav1.Now(),
			})
			meta.SetStatusCondition(&tb.Status.Conditions, metav1.Condition{
				Type:               typeDegraded,
				Status:             metav1.ConditionFalse,
				Reason:             "AllReplicasReady",
				Message:            "Cluster is healthy",
				LastTransitionTime: metav1.Now(),
			})
		} else if sts.Status.ReadyReplicas > 0 {
			tb.Status.Phase = "Degraded"
			meta.SetStatusCondition(&tb.Status.Conditions, metav1.Condition{
				Type:               typeDegraded,
				Status:             metav1.ConditionTrue,
				Reason:             "SomeReplicasNotReady",
				Message:            fmt.Sprintf("%d of %d replicas ready", sts.Status.ReadyReplicas, *sts.Spec.Replicas),
				LastTransitionTime: metav1.Now(),
			})
		} else {
			tb.Status.Phase = "Pending"
		}
	}

	return r.Status().Update(ctx, tb)
}

// SetupWithManager sets up the controller with the Manager.
func (r *TigerBeetleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.TigerBeetle{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Named("tigerbeetle").
		Complete(r)
}
