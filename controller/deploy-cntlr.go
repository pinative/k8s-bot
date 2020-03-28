package controller

import (
	"fmt"
	"github.com/pinative/k8s-bot/pkg/helper"
	"github.com/pinative/k8s-bot/pkg/service"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	informerappsv1 "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"os"
)

type DeploymentController struct {
	informerFactory    informers.SharedInformerFactory
	deploymentInformer informerappsv1.DeploymentInformer
}

func (c *DeploymentController) Sync(stopCh <-chan struct{}) error {
	// Starts all the shared informers that have been created by the factory so far.
	c.informerFactory.Start(stopCh)

	// wait for the initial synchronization of the local cache.
	if !cache.WaitForCacheSync(stopCh, c.deploymentInformer.Informer().HasSynced) {
		log.Error().Msg("failed to sync deployment data")
		return fmt.Errorf("failed to sync deployment data")
	}
	return nil
}

// Run starts shared informers and waits for the shared controller cache to
//  synchronize.
func (c *DeploymentController) Run(stopCh <-chan struct{}) {
	c.deploymentInformer.Informer().Run(stopCh)
}

func (c *DeploymentController) onAddFunc(obj interface{}) {
	deploy := obj.(*appsv1.Deployment)
	ns := deploy.Namespace

	flag := helper.AreNamespaceInExcludesList(ns, ExcludesNamespaceList)
	if flag {
		return
	}

	log.Printf("DEPLOYMENT %s/%s was CREATED at %v", deploy.GetNamespace(), deploy.Name, deploy.CreationTimestamp)
}

func (c *DeploymentController) onUpdateFunc(old, new interface{}) {
	oldDeploy := old.(*appsv1.Deployment)
	newDeploy := new.(*appsv1.Deployment)

	flag := helper.AreNamespaceInExcludesList(oldDeploy.GetNamespace(), ExcludesNamespaceList)
	if flag {
		return
	}

	log.Printf("DEPLOYMENT %s/%s was UPDATED", newDeploy.Namespace, newDeploy.Name)

	if newDeploy.Annotations["pigo.io/part-of"] == os.Getenv("ANNOT_PIGO_IO_PARTOF") {
		ol := oldDeploy.GetLabels()
		ns := oldDeploy.GetNamespace()
		svc := &service.Service{
			K8sClient: helper.GetClientset(),
			Namespace: ns,
		}
		_ = svc.UpsertService(c.informerFactory, ol, newDeploy, oldDeploy)
	}
}

func (c *DeploymentController) onDeleteFunc(obj interface{}) {
	deploy := obj.(*appsv1.Deployment)

	flag := helper.AreNamespaceInExcludesList(deploy.GetNamespace(), ExcludesNamespaceList)
	if flag || deploy.DeletionTimestamp != nil {
		return
	}

	log.Printf("DEPLOYMENT %s/%s was DELETED at %v", deploy.Namespace, deploy.Name, deploy.DeletionTimestamp)
	if deploy != nil && deploy.Annotations["pigo.io/part-of"] == os.Getenv("ANNOT_PIGO_IO_PARTOF") {
		l := deploy.GetLabels()
		ns := deploy.GetNamespace()
		svc := &service.Service{
			K8sClient: helper.GetClientset(),
			Namespace: ns,
		}
		_ = svc.DeleteService(c.informerFactory, l)
	}
}

func NewDeploymentController(informerFactory informers.SharedInformerFactory) *DeploymentController {
	deployInformer := informerFactory.Apps().V1().Deployments()

	dc := &DeploymentController{
		informerFactory:    informerFactory,
		deploymentInformer: deployInformer,
	}
	deployInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    dc.onAddFunc,
			UpdateFunc: dc.onUpdateFunc,
			DeleteFunc: dc.onDeleteFunc,
		},
	)

	return dc
}
