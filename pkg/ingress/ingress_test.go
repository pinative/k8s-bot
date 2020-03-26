package ingress

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	"log"
	"os"
	"testing"
)

func init() {
	log.Println("Setting environment variable for Ingress testing")
	_ = os.Setenv("BOT_SERVICE_PREFIX", "svc-")
	_ = os.Setenv("BOT_INGRESS_PREFIX", "ing-")
}

func newService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: os.Getenv("BOT_SERVICE_PREFIX") + "fake-test",
			Namespace: "fake-test",
		},
	}
}

func newFakeNetworkingIngress() *v1beta1.Ingress {
	return &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: os.Getenv("BOT_INGRESS_PREFIX") + "fake-test",
			Namespace: "fake-test",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "fake-test.apps.pidns.host",
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Backend: v1beta1.IngressBackend{
										ServiceName: os.Getenv("BOT_SERVICE_PREFIX") + "fake-test",
										ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: 80 },
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func newFakeIngress() *Ingress {
	return &Ingress{
		K8sClient: fake.NewSimpleClientset(newService(), newFakeNetworkingIngress()),
	}
}

func TestIngress_CreateIngress(t *testing.T) {
	ing := newFakeIngress()
	sn := os.Getenv("BOT_SERVICE_PREFIX") + "fake-create-new"
	ns := "fake-test"
	err := ing.CreateIngress(sn, ns, []corev1.ServicePort{{Name: os.Getenv("BOT_SERVICE_PREFIX") + "port-fake-create-new", Port: int32(80), TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(8080)}}})
	if err != nil {
		t.Errorf("Expected without any error to create a new ingress, but got error: %v", err)
		return
	}

	ingress, err := ing.K8sClient.NetworkingV1beta1().Ingresses(ns).Get(getIngressName(sn), metav1.GetOptions{})
	if err != nil {
		t.Errorf("Expected get the ingress just created, but got error: %v", err)
		return
	}

	if ingress.Name != getIngressName(sn) {
		t.Errorf("Expected the created ingress name to be %s, but got %s", getIngressName(sn), ingress.Name)
	}
}

func TestIngress_UpdateIngress(t *testing.T) {
	ing := newFakeIngress()
	ing.ServiceName = os.Getenv("BOT_SERVICE_PREFIX") + "fake-test"
	ing.Namespace = "fake-test"

	ingresses := []*v1beta1.Ingress{newFakeNetworkingIngress()}

	nsn := os.Getenv("BOT_SERVICE_PREFIX") + "fake-new-test"
	nsp := int32(8080)
	err := ing.UpdateIngress(ingresses, ing.ServiceName, nsn, ing.Namespace, nsp)
	if err != nil {
		t.Errorf("Expected no error thrown when updating ingress by service name %s, but got error: %v", ing.ServiceName, err)
		return
	}

	i, _ := ing.K8sClient.NetworkingV1beta1().Ingresses(ing.Namespace).Get(getIngressName(ing.ServiceName), metav1.GetOptions{})
	if len(i.Spec.Rules) > 1 {
		t.Errorf("Expected only have one rule for the ingress %s, but got %v", getIngressName(ing.ServiceName), len(i.Spec.Rules))
	}

	if len(i.Spec.Rules[0].HTTP.Paths) > 1 {
		t.Errorf("Expected only have one path for the ingress %s, but got %v", getIngressName(ing.ServiceName), len(i.Spec.Rules[0].HTTP.Paths))
	}

	un := i.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName
	if un != nsn {
		t.Errorf("Expected the service name should be updated, but got the old service name %s.", un)
	}

	up := i.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort
	if up.IntVal == 80 || up.IntVal != nsp {
		t.Errorf("Expected the service port should be updated to %v, but got %v", nsp, up.IntVal)
	}
}

func TestIngress_DeleteIngressWithPass(t *testing.T) {
	ing := newFakeIngress()
	ing.ServiceName = os.Getenv("BOT_SERVICE_PREFIX") + "fake-test"
	ing.Namespace = "fake-test"

	err := ing.DeleteIngress()
	if err != nil {
		t.Errorf("Expected without any errors for deleting the ingress by service name %s, but got error: %v", ing.ServiceName, err)
	}
}

func TestIngress_DeleteIngressWithError(t *testing.T) {
	ing := newFakeIngress()
	ing.ServiceName = os.Getenv("BOT_SERVICE_PREFIX") + "test"
	ing.Namespace = "fake-test"

	err := ing.DeleteIngress()
	if err == nil {
		t.Errorf("Expected an error for deleting the ingress by service name %s, but no errors thrown", ing.ServiceName)
	}
}

func TestHasIngressExists(t *testing.T) {
	ingresses := []*v1beta1.Ingress{
		getFakeIngress(),
	}

	isExisted := HasIngressExists(os.Getenv("BOT_SERVICE_PREFIX") + "fake-test", ingresses)
	if !isExisted {
		t.Errorf("Expected the ingress exist status to be true but got %v", isExisted)
	}

	isExisted = HasIngressExists("fake-test", ingresses)
	if isExisted {
		t.Errorf("Expected the ingress exist status to be false but got %v", isExisted)
	}
}

func getFakeIngress() (ing *v1beta1.Ingress) {
	httpIrv := v1beta1.HTTPIngressRuleValue{
		Paths: []v1beta1.HTTPIngressPath{
			{
				Backend: v1beta1.IngressBackend{
					ServiceName: os.Getenv("BOT_SERVICE_PREFIX") + "fake-test",
					ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: 80},
				},
			},
		},
	}
	irv := v1beta1.IngressRuleValue{
		HTTP: &httpIrv,
	}
	ing = &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: os.Getenv("BOT_INGRESS_PREFIX") + "fake-test",
			Namespace: "fake-test",
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: "test.apps.pidns.host",
					IngressRuleValue: irv,
				},
			},
		},
	}
	return
}