package controller

import (
	"fmt"
	"github.com/pinative/k8s-bot/pkg/helper"
	"github.com/rs/zerolog/log"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/client-go/informers"
	informernetv1beta1 "k8s.io/client-go/informers/networking/v1beta1"
	"k8s.io/client-go/tools/cache"
)

type IngressController struct {
	informerFactory informers.SharedInformerFactory
	ingressInformer     informernetv1beta1.IngressInformer
}

func (c *IngressController) Sync(stopCh <-chan struct{}) error {
	// Starts all the shared informers that have been created by the factory so far.
	c.informerFactory.Start(stopCh)

	// wait for the initial synchronization of the local cache.
	if !cache.WaitForCacheSync(stopCh, c.ingressInformer.Informer().HasSynced) {
		log.Error().Msg("failed to sync ingress data")
		return fmt.Errorf("failed to sync ingress data")
	}
	return nil
}

// Run starts shared informers and waits for the shared controller cache to
//  synchronize.
func (c *IngressController) Run(stopCh <-chan struct{}) {
	c.ingressInformer.Informer().Run(stopCh)
}

func (c *IngressController) onAddFunc(obj interface{}) {
	ing := obj.(*networkingv1beta1.Ingress)

	flag := helper.AreNamespaceInExcludesList(ing.Namespace, ExcludesNamespaceList)
	if flag {
		return
	}

	log.Printf("INGRESS %s/%s was CREATED at %v", ing.Namespace, ing.Name, ing.CreationTimestamp)
}

func (c *IngressController) onUpdateFunc(old, new interface{}) {
	oldIng := old.(*networkingv1beta1.Ingress)
	newIng := new.(*networkingv1beta1.Ingress)

	flag := helper.AreNamespaceInExcludesList(oldIng.Namespace, ExcludesNamespaceList)
	if flag {
		return
	}

	if oldIng == newIng {
		log.Printf("INGRESS %s/%s was UPDATED.", oldIng.Namespace, oldIng.Name)
	}
}

func (c *IngressController) onDeleteFunc(obj interface{}) {
	ing := obj.(*networkingv1beta1.Ingress)

	flag := helper.AreNamespaceInExcludesList(ing.Namespace, ExcludesNamespaceList)
	if flag {
		return
	}

	log.Printf("INGRESS %s/%s was DELETED at %v", ing.Namespace, ing.Name, ing.DeletionTimestamp)
}

func NewIngressController(informerFactory informers.SharedInformerFactory) *IngressController {
	ingressInformer := informerFactory.Networking().V1beta1().Ingresses()

	ic := &IngressController{
		informerFactory:   informerFactory,
		ingressInformer:   ingressInformer,
	}
	ingressInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: ic.onAddFunc,
			UpdateFunc: ic.onUpdateFunc,
			DeleteFunc: ic.onDeleteFunc,
		},
	)

	return ic
}
