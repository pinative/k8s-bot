package controller

import (
	"fmt"
	"github.com/pinative/k8s-bot/helper"
	"github.com/pinative/k8s-bot/pkg/ingress"
	"github.com/rs/zerolog/log"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	informersv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	"os"
)

type ServiceController struct {
	informerFactory informers.SharedInformerFactory
	serviceInformer     informersv1.ServiceInformer
}

func (c *ServiceController) Sync(stopCh <-chan struct{}) error {
	// Starts all the shared informers that have been created by the factory so far.
	c.informerFactory.Start(stopCh)

	// wait for the initial synchronization of the local cache.
	if !cache.WaitForCacheSync(stopCh, c.serviceInformer.Informer().HasSynced) {
		log.Error().Msg("failed to sync service data")
		return fmt.Errorf("failed to sync service data")
	}
	return nil
}

// Run starts shared informers and waits for the shared controller cache to
//  synchronize.
func (c *ServiceController) Run(stopCh <-chan struct{}) {
	c.serviceInformer.Informer().Run(stopCh)
}

func (c *ServiceController) onAddFunc(obj interface{}) {
	//svc := obj.(metav1.Object)
	svc := obj.(*v1.Service)
	flag := helper.AreNamespaceInWhiteList(svc.GetNamespace(), ExcludesNamespaceList)
	if flag {
		return
	}

	log.Printf("SERVICE %s/%s was CREATED at %v", svc.GetNamespace(), svc.GetName(), svc.GetCreationTimestamp())
	ing := ingress.Ingress{
		K8sClient: helper.GetClientset(),
	}
	_ = ing.UpsertIngress(svc, nil, c.informerFactory)
	//ret, err := ingress.getIngresses(svc.Name, svc.Namespace, c.informerFactory)
	//if err != nil {
	//	return
	//}
	//
	//annots := svc.GetAnnotations()
	//aia := annots["pigo.network/allow-internet-access"]
	//if annots["pigo.io/part-of"] == os.Getenv("ANNOT_PIGO_IO_PARTOF") && aia == "true" && !ingress.HasIngressExists(svc.GetName(), ret) {
	//	ingress.CreateIngress(svc.Name, svc.Namespace, svc.Spec.Ports)
	//}
}

func (c *ServiceController) onUpdateFunc(old, new interface{}) {
	oldSvc := old.(*v1.Service)
	newSvc := new.(*v1.Service)
	log.Printf("SERVICE %s/%s was UPDATED", newSvc.Namespace, newSvc.Name)
	ing := ingress.Ingress{
		K8sClient: helper.GetClientset(),
	}
	_ = ing.UpsertIngress(newSvc, oldSvc, c.informerFactory)
}


func (c *ServiceController) onDeleteFunc(obj interface{}) {
	svc := obj.(*v1.Service)
	if svc.Annotations["pigo.io/part-of"] == os.Getenv("ANNOT_PIGO_IO_PARTOF") {
		log.Printf("SERVICE %s/%s was DELETED at %v", svc.Namespace, svc.Name, svc.DeletionTimestamp)
		ing := ingress.Ingress{
			K8sClient: helper.GetClientset(),
			ServiceName: svc.Name,
			Namespace: svc.Namespace,
		}
		_ = ing.DeleteIngress()
	}
}

func NewServiceController(informerFactory informers.SharedInformerFactory) *ServiceController {
	svcInformer := informerFactory.Core().V1().Services()

	sc := &ServiceController{
		informerFactory: informerFactory,
		serviceInformer:     svcInformer,
	}
	svcInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    sc.onAddFunc,
			UpdateFunc: sc.onUpdateFunc,
			DeleteFunc: sc.onDeleteFunc,
		},
	)

	return sc
}
