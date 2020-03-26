package controller

import (
	"context"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func newFakeDeployment() *v1.Deployment {
	return &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-deploy-name",
			Namespace: "fake-test",
		},
	}
}

func TestDeploymentController_Sync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fd := newFakeDeployment()
	client := fake.NewSimpleClientset(fd)
	isf := informers.NewSharedInformerFactory(client, 0)
	dc := NewDeploymentController(isf)

	_ = dc.Sync(ctx.Done())

	d, err := dc.deploymentInformer.Lister().Deployments(fd.Namespace).Get(fd.Name)
	if err != nil {
		t.Errorf("Expected no errors occured to list deployment %s from informer lister, but got error: %v", fd.Name, err)
	}

	if d == nil {
		t.Errorf("Expected returns a deployment, but no deployment returned")
	}
}
