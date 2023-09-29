/*
 *
 * Copyright 2023 @ Linying Assad <linying@apache.org>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 * /
 */

package reconciler

import (
	"fmt"
	dapi "github.com/al-assad/doris-operator/api/v1beta1"
	"github.com/al-assad/doris-operator/internal/transformer"
	"github.com/al-assad/doris-operator/internal/util"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	FeConfHashAnnotationKey     = fmt.Sprintf("%s/fe-config", dapi.GroupVersion.Group)
	BeConfHashAnnotationKey     = fmt.Sprintf("%s/be-config", dapi.GroupVersion.Group)
	CnConfHashAnnotationKey     = fmt.Sprintf("%s/cn-config", dapi.GroupVersion.Group)
	BrokerConfHashAnnotationKey = fmt.Sprintf("%s/broker-config", dapi.GroupVersion.Group)
)

// DorisClusterReconciler reconciles a DorisCluster object
type DorisClusterReconciler struct {
	ReconcileContext
	CR *dapi.DorisCluster
}

// ClusterStageRecResult represents the result of a stage reconciliation for DorisCluster
type ClusterStageRecResult struct {
	Stage  dapi.DorisClusterOprStage
	Status dapi.OprStageStatus
	Action dapi.OprStageAction
	Err    error
}

func clusterStageSucc(stage dapi.DorisClusterOprStage, action dapi.OprStageAction) ClusterStageRecResult {
	return ClusterStageRecResult{Stage: stage, Status: dapi.StageResultSucceeded, Action: action}
}

func clusterStageFail(stage dapi.DorisClusterOprStage, action dapi.OprStageAction, err error) ClusterStageRecResult {
	return ClusterStageRecResult{Stage: stage, Status: dapi.StageResultSucceeded, Action: action, Err: err}
}

// Reconcile secret object that using to store the sql query account info
// that used by doris-operator.
func (r *DorisClusterReconciler) recOprAccountSecret() ClusterStageRecResult {
	secretRef := transformer.GetOprSqlAccountSecretName(r.CR)
	action := dapi.StageActionApply
	// check if secret exists
	exists, err := r.Exist(secretRef, &corev1.Secret{})
	if err != nil {
		return clusterStageFail(dapi.StageSqlAccountSecret, action, err)
	}
	// create secret if not exists
	if !exists {
		newSecret := transformer.MakeOprSqlAccountSecret(r.CR)
		if err := r.Create(r.Ctx, newSecret); err != nil {
			return clusterStageFail(dapi.StageSqlAccountSecret, action, err)
		}
	}
	return clusterStageSucc(dapi.StageSqlAccountSecret, action)
}

// Reconcile Doris FE component resources.
func (r *DorisClusterReconciler) recFeResources() ClusterStageRecResult {
	if r.CR.Spec.FE != nil {
		// apply resources
		action := dapi.StageActionApply
		// fe configmap
		configMap := transformer.MakeFeConfigMap(r.CR, r.Schema)
		if err := r.CreateOrUpdate(configMap); err != nil {
			return clusterStageFail(dapi.StageFeConfigmap, action, err)
		}
		// fe service
		service := transformer.MakeFeService(r.CR, r.Schema)
		if err := r.CreateOrUpdate(service); err != nil {
			return clusterStageFail(dapi.StageFeService, action, err)
		}
		peerService := transformer.MakeFePeerService(r.CR, r.Schema)
		if err := r.CreateOrUpdate(peerService); err != nil {
			return clusterStageFail(dapi.StageFeService, action, err)
		}
		// fe statefulset
		statefulSet := transformer.MakeFeStatefulSet(r.CR, r.Schema)
		statefulSet.Annotations[FeConfHashAnnotationKey] = util.MapMd5(configMap.Data)
		if err := r.CreateOrUpdate(statefulSet); err != nil {
			return clusterStageFail(dapi.StageFeStatefulSet, action, err)
		}
		return clusterStageSucc(dapi.StageFe, action)
	} else {
		// delete resources
		action := dapi.StageActionDelete
		// fe configmap
		configMapRef := transformer.GetFeConfigMapName(r.CR)
		if err := r.DeleteWhenExist(configMapRef, &corev1.ConfigMap{}); err != nil {
			return clusterStageFail(dapi.StageFeConfigmap, action, err)
		}
		// fe service
		serviceRef := transformer.GetFeServiceName(r.CR)
		if err := r.DeleteWhenExist(serviceRef, &corev1.Service{}); err != nil {
			return clusterStageFail(dapi.StageFeConfigmap, action, err)
		}
		peerServiceRef := transformer.GetFePeerServiceName(r.CR)
		if err := r.DeleteWhenExist(peerServiceRef, &corev1.Service{}); err != nil {
			return clusterStageFail(dapi.StageFeConfigmap, action, err)
		}
		// fe statefulset
		statefulsetRef := transformer.GetFeStatefulSetName(r.CR)
		if err := r.DeleteWhenExist(statefulsetRef, &appv1.StatefulSet{}); err != nil {
			return clusterStageFail(dapi.StageFeConfigmap, action, err)
		}
		return clusterStageSucc(dapi.StageFe, action)
	}
}

// Reconcile Doris BE component resources.
func (r *DorisClusterReconciler) recBeResources() ClusterStageRecResult {
	if r.CR.Spec.BE != nil {
		// apply resources
		action := dapi.StageActionApply
		// be configmap
		configMap := transformer.MakeBeConfigMap(r.CR, r.Schema)
		if err := r.CreateOrUpdate(configMap); err != nil {
			return clusterStageFail(dapi.StageBeConfigmap, action, err)
		}
		// be service
		service := transformer.MakeBeService(r.CR, r.Schema)
		if err := r.CreateOrUpdate(service); err != nil {
			return clusterStageFail(dapi.StageBeService, action, err)
		}
		peerService := transformer.MakeBePeerService(r.CR, r.Schema)
		if err := r.CreateOrUpdate(peerService); err != nil {
			return clusterStageFail(dapi.StageBeService, action, err)
		}
		// be statefulset
		statefulSet := transformer.MakeBeStatefulSet(r.CR, r.Schema)
		statefulSet.Annotations[BeConfHashAnnotationKey] = util.MapMd5(configMap.Data)
		if err := r.CreateOrUpdate(statefulSet); err != nil {
			return clusterStageFail(dapi.StageBeStatefulSet, action, err)
		}
		return clusterStageSucc(dapi.StageBe, action)
	} else {
		// delete resources
		action := dapi.StageActionDelete
		// be configmap
		configMapRef := transformer.GetBeConfigMapName(r.CR)
		if err := r.DeleteWhenExist(configMapRef, &corev1.ConfigMap{}); err != nil {
			return clusterStageFail(dapi.StageBeConfigmap, action, err)
		}
		// be service
		serviceRef := transformer.GetBeServiceName(r.CR)
		if err := r.DeleteWhenExist(serviceRef, &corev1.Service{}); err != nil {
			return clusterStageFail(dapi.StageBeConfigmap, action, err)
		}
		peerServiceRef := transformer.GetBePeerServiceName(r.CR)
		if err := r.DeleteWhenExist(peerServiceRef, &corev1.Service{}); err != nil {
			return clusterStageFail(dapi.StageBeConfigmap, action, err)
		}
		// be statefulset
		statefulsetRef := transformer.GetBeStatefulSetName(r.CR)
		if err := r.DeleteWhenExist(statefulsetRef, &appv1.StatefulSet{}); err != nil {
			return clusterStageFail(dapi.StageBeConfigmap, action, err)
		}
		return clusterStageSucc(dapi.StageBe, action)
	}
}

// Reconcile Doris CN component resources.
func (r *DorisClusterReconciler) recCnResources() ClusterStageRecResult {
	if r.CR.Spec.CN != nil {
		// apply resources
		action := dapi.StageActionApply
		// cn configmap
		configMap := transformer.MakeCnConfigMap(r.CR, r.Schema)
		if err := r.CreateOrUpdate(configMap); err != nil {
			return clusterStageFail(dapi.StageCnConfigmap, action, err)
		}
		// cn service
		service := transformer.MakeCnService(r.CR, r.Schema)
		if err := r.CreateOrUpdate(service); err != nil {
			return clusterStageFail(dapi.StageCnService, action, err)
		}
		peerService := transformer.MakeCnPeerService(r.CR, r.Schema)
		if err := r.CreateOrUpdate(peerService); err != nil {
			return clusterStageFail(dapi.StageCnService, action, err)
		}

		// cn statefulset
		statefulSet := transformer.MakeCnStatefulSet(r.CR, r.Schema)
		statefulSet.Annotations[CnConfHashAnnotationKey] = util.MapMd5(configMap.Data)
		// todo when the corresponding DorisAutoScaler resource exists,
		// the replica of statefulset would not be overridden

		if err := r.CreateOrUpdate(statefulSet); err != nil {
			return clusterStageFail(dapi.StageCnStatefulSet, action, err)
		}
		return clusterStageSucc(dapi.StageCn, action)
	} else {
		// delete resources
		action := dapi.StageActionDelete
		// cn configmap
		configMapRef := transformer.GetCnConfigMapName(r.CR)
		if err := r.DeleteWhenExist(configMapRef, &corev1.ConfigMap{}); err != nil {
			return clusterStageFail(dapi.StageCnConfigmap, action, err)
		}
		// cn service
		serviceRef := transformer.GetCnServiceName(r.CR)
		if err := r.DeleteWhenExist(serviceRef, &corev1.Service{}); err != nil {
			return clusterStageFail(dapi.StageCnConfigmap, action, err)
		}
		peerServiceRef := transformer.GetCnPeerServiceName(r.CR)
		if err := r.DeleteWhenExist(peerServiceRef, &corev1.Service{}); err != nil {
			return clusterStageFail(dapi.StageCnConfigmap, action, err)
		}
		// cn statefulset
		statefulsetRef := transformer.GetCnStatefulSetName(r.CR)
		if err := r.DeleteWhenExist(statefulsetRef, &appv1.StatefulSet{}); err != nil {
			return clusterStageFail(dapi.StageCnConfigmap, action, err)
		}
		return clusterStageSucc(dapi.StageCn, action)
	}
}
