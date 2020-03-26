package service

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"log"
	"os"
	"reflect"
	"testing"
)

func init() {
	log.Println("Setting environment variable for Service testing")
	_ = os.Setenv("BOT_SERVICE_PREFIX", "svc-")
	_ = os.Setenv("BOT_INGRESS_PREFIX", "ing-")
}

func newFakeService() *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: os.Getenv("BOT_SERVICE_PREFIX") + "fake-service",
			Namespace: "fake-test",
			Labels: map[string]string{"test-service": "true"},
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: "svc-port-fake-test",
					Port: int32(80),
				},
			},
		},
	}
}

func TestService_DeleteService(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset(newFakeService())
	infmrs := informers.NewSharedInformerFactory(client, 0)
	svcInformer := infmrs.Core().V1().Services().Informer()
	infmrs.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), svcInformer.HasSynced)

	fakeSvc := Service{
		K8sClient: client,
		Namespace: "fake-test",
	}
	l := map[string]string{"test-service": "true"}

	err := fakeSvc.DeleteService(infmrs, l)
	if err != nil {
		t.Errorf("Expected without any error to delete the service by labels %v", l)
	}

	sl, err := fakeSvc.K8sClient.CoreV1().Services(fakeSvc.Namespace).List(metav1.ListOptions{})
	if err != nil {
		t.Errorf("Expected no error occurs to list services from informers, but got error %v", err)
	}

	if len(sl.Items) > 0 {
		t.Errorf("Expected empty services, but got %v", len(sl.Items))
	}
}

func TestService_UpsertServiceWithServiceCreation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset(newFakeService())
	infmrs := informers.NewSharedInformerFactory(client, 0)
	svcInformer := infmrs.Core().V1().Services().Informer()
	infmrs.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), svcInformer.HasSynced)

	fakeSvc := Service{
		K8sClient: client,
		Namespace: "fake-test",
	}

	ol := map[string]string{"test-new-service": "true"}
	nd := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-deploy-name",
			Namespace: "fake-test",
			Labels: map[string]string{"fake-label-key": "fake-value"},
			Annotations: map[string]string{"pigo.network/allow-internet-access": "false"},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}
	od := &appsv1.Deployment{}
	err := fakeSvc.UpsertService(infmrs, ol, nd, od)
	if err != nil {
		t.Errorf("Expected no errors occured to update the service, but got error: %v", err)
	}

	svcName := os.Getenv("BOT_SERVICE_PREFIX") + nd.Name
	svc, err := fakeSvc.K8sClient.CoreV1().Services(fakeSvc.Namespace).Get(svcName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Expected no errors to get the service %s, but got error: %v", svcName, err)
	}
	t.Logf("svc returned %v", svc)

	if svc == nil {
		t.Errorf("Expected returns the created service, but got nil")
	}
}

func TestService_UpsertServiceWithServiceUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fs := newFakeService()
	client := fake.NewSimpleClientset(fs)
	infmrs := informers.NewSharedInformerFactory(client, 0)
	svcInformer := infmrs.Core().V1().Services().Informer()
	infmrs.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), svcInformer.HasSynced)

	fakeSvc := Service{
		K8sClient: client,
		Namespace: "fake-test",
	}

	ol := map[string]string{"test-service": "true"}
	nd := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"fake-new-label-key": "fake-value"},
			ResourceVersion: "25654644",
		},
	}
	od := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "25654622",
		},
	}
	err := fakeSvc.UpsertService(infmrs, ol, nd, od)
	if err != nil {
		t.Errorf("Expected no errors occured to update the service, but got error: %v", err)
	}

	svc, err := fakeSvc.K8sClient.CoreV1().Services(fakeSvc.Namespace).Get(fs.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("Expected no errors to get the service %s, but got error: %v", fs.Name, err)
	}

	if !reflect.DeepEqual(svc.GetLabels(), nd.GetLabels()) {
		t.Errorf("Expected returns labels %v from the new service, but got %v", nd.Labels, svc.GetLabels())
	}
}

func TestService_UpsertServiceWithEmptyLabels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset(newFakeService())
	infmrs := informers.NewSharedInformerFactory(client, 0)
	svcInformer := infmrs.Core().V1().Services().Informer()
	infmrs.Start(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), svcInformer.HasSynced)

	fakeSvc := Service{
		K8sClient: client,
		Namespace: "fake-test",
	}

	ol := map[string]string{}
	nd := &appsv1.Deployment{}
	od := &appsv1.Deployment{}
	err := fakeSvc.UpsertService(infmrs, ol, nd, od)
	if err == nil {
		t.Errorf("Expected an invalid arguments error to be returned, but did not.")
	}
}

func TestGetServicePortWithoutError(t *testing.T) {
	sn := "svc-fake-test"
	ports := []v1.ServicePort{
		{
			Name: "svc-port-fake-test",
			Port: int32(80),
		},
	}

	p := GetServicePort(sn, ports)
	if p != 80 {
		t.Errorf("Expected service port to be 80, but got %v", p)
	}
}

func TestGetServicePortWithError(t *testing.T) {
	sn := "svc-fake-svc-test"
	ports := []v1.ServicePort{
		{
			Name: "svc-port-fake-test",
			Port: int32(80),
		},
	}

	p := GetServicePort(sn, ports)
	t.Logf("port returned %v", p)
	if p == 80 {
		t.Errorf("Expected service port to be 0, but got %v", p)
	}
}