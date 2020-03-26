package controller

import (
	"context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func newFakeService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-ing-name",
			Namespace: "fake-test",
		},
	}
}

func TestServiceController_Sync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs := newFakeService()
	client := fake.NewSimpleClientset(fs)
	isf := informers.NewSharedInformerFactory(client, 0)
	dc := NewServiceController(isf)

	_ = dc.Sync(ctx.Done())

	d, err := dc.serviceInformer.Lister().Services(fs.Namespace).Get(fs.Name)
	if err != nil {
		t.Errorf("Expected no errors occured to list service %s from informer lister, but got error: %v", fs.Name, err)
	}

	if d == nil {
		t.Errorf("Expected returns a deployment, but no deployment returned")
	}
}
