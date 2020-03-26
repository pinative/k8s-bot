package controller

import (
	"context"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func newFakeIngress() *v1beta1.Ingress {
	return &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-ing-name",
			Namespace: "fake-test",
		},
	}
}

func TestIngressController_Sync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fi := newFakeIngress()
	client := fake.NewSimpleClientset(fi)
	isf := informers.NewSharedInformerFactory(client, 0)
	dc := NewIngressController(isf)

	_ = dc.Sync(ctx.Done())

	d, err := dc.ingressInformer.Lister().Ingresses(fi.Namespace).Get(fi.Name)
	if err != nil {
		t.Errorf("Expected no errors occured to list ingress %s from informer lister, but got error: %v", fi.Name, err)
	}

	if d == nil {
		t.Errorf("Expected returns a deployment, but no deployment returned")
	}
}
