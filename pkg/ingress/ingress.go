package ingress

import (
	"encoding/json"
	"github.com/pinative/k8s-bot/helper"
	"github.com/pinative/k8s-bot/pkg/service"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"os"
	"strings"
)

type Ingress struct {
	ServiceName string
	Namespace string
	K8sClient kubernetes.Interface
}

func (i *Ingress) UpsertIngress(newSvc *corev1.Service, oldSvc *corev1.Service, iif informers.SharedInformerFactory) (err error) {
	annots := newSvc.GetAnnotations()
	aia := annots["pigo.network/allow-internet-access"]
	if annots["pigo.io/part-of"] == os.Getenv("ANNOT_PIGO_IO_PARTOF") && aia == "true" {
		var (
			ingresses []*networkingv1beta1.Ingress
			err error
			on string
			oss corev1.ServiceSpec

			nn string
			nns string
			nss corev1.ServiceSpec
		)
		if oldSvc != nil {
			ingresses, err = getIngresses(oldSvc.Name, oldSvc.Namespace, iif)
			on = oldSvc.Name
			oss = oldSvc.Spec
		} else if newSvc != nil {
			ingresses, err = getIngresses(newSvc.Name, newSvc.Namespace, iif)
			nn = newSvc.Name
			nns = newSvc.Namespace
			nss = newSvc.Spec
		}
		if err != nil {
			return err
		}

		isExisted := HasIngressExists(on, ingresses)
		if isExisted {
			// If the service name or service port has been changed
			//  then update corresponding ingress if it exists
			osp := service.GetServicePort(on, oss.Ports)
			nsp := service.GetServicePort(nn, nss.Ports)
			if on != nn ||
				(osp != 0 && nsp != 0 && osp != nsp) {
				err = i.UpdateIngress(ingresses, on, nn, nns, nsp)
			}
		} else {
			err = i.CreateIngress(nn, nns, nss.Ports)
		}
	}

	return
}

func (i *Ingress) CreateIngress(sn, ns string, sp []corev1.ServicePort) (err error) {
	ing := newIngress(helper.GetPublicDns(), sn, ns, sp)
	_, err = i.K8sClient.NetworkingV1beta1().Ingresses(ns).Create(ing)
	if err != nil {
		log.Error().
			Err(err).
			Str("namespace", ns).
			Str("ingress name", ing.Name).
			Send()
	}

	return
}

func (i *Ingress) UpdateIngress(ingresses []*networkingv1beta1.Ingress, osn, nsn, ns string, nsp int32) (err error) {
	ingName := getIngressName(osn)
	ingress := networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: ingName},
		Spec: networkingv1beta1.IngressSpec{
			Rules: []networkingv1beta1.IngressRule{
				{
					IngressRuleValue: networkingv1beta1.IngressRuleValue{
						HTTP: &networkingv1beta1.HTTPIngressRuleValue{
							Paths: []networkingv1beta1.HTTPIngressPath{
								{
									Backend: networkingv1beta1.IngressBackend{},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, ing := range ingresses {
		if ing.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName == osn {
			if nsn != "" {
				ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName = nsn
			}

			if nsp != 0 {
				ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort = intstr.IntOrString{Type: intstr.Int, IntVal: nsp}
			}
			break
		}
	}

	if nsn != "" {
		ij, err := json.Marshal(ingress)
		if err != nil {
			log.Error().
				Err(err).
				Str("namespace", ns).
				Str("ingress name", ingName).
				Msg("marshal ingress")
		}
		_, err = i.K8sClient.NetworkingV1beta1().Ingresses(ns).Patch(ingName, types.StrategicMergePatchType, ij)
		if err != nil {
			log.Error().
				Err(err).
				Str("namespace", ns).
				Str("ingress name", ingName).
				Msgf("patch ingress %v", string(ij))
		}

	}

	return err
}

func (i *Ingress) DeleteIngress() (err error) {
	deletePolicy := metav1.DeletePropagationForeground
	ingName := getIngressName(i.ServiceName)
	log.Info().Str("ingName", ingName)
	ns := i.Namespace
	err = i.K8sClient.NetworkingV1beta1().Ingresses(ns).Delete(ingName, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("namespace", ns).
			Str("ingress name", ingName).
			Send()
	}

	return
}

func HasIngressExists(sn string, ingresses []*networkingv1beta1.Ingress) bool {
	for _, ing := range ingresses {
		for _, ir := range ing.Spec.Rules {
			for _, p := range ir.HTTP.Paths {
				if sn == p.Backend.ServiceName {
					return true
				}
			}
		}
	}

	return false
}

func getIngresses(sn, ns string, informer informers.SharedInformerFactory) (ret []*networkingv1beta1.Ingress, err error) {
	ingLister := informer.Networking().V1beta1().Ingresses()
	ret, err = ingLister.Lister().Ingresses(sn).List(labels.Everything())
	if err != nil {
		log.Error().Err(err).Msgf("onAddFunc - Error to list ingresses by labels %v from namespace %s", labels.Everything(), ns)
	}

	return
}

func newIngress(hostname, sn, ns string, sp []corev1.ServicePort) (ing *networkingv1beta1.Ingress) {
	annotations := map[string]string{"nginx.ingress.kubernetes.io/rewrite-target": "/"}

	ing = &networkingv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        getIngressName(sn),
			Namespace:   ns,
			Annotations: annotations,
		},
		Spec: networkingv1beta1.IngressSpec{
			Rules: getRules(sn, hostname, sp),
		},
	}

	return
}

func getIngressName(sn string) string {
	return os.Getenv("BOT_INGRESS_PREFIX") + strings.TrimPrefix(sn, os.Getenv("BOT_SERVICE_PREFIX"))
}

func getPaths(sn string, ports []corev1.ServicePort) (paths []networkingv1beta1.HTTPIngressPath) {
	var port int32
	for _, sp := range ports {
		if strings.HasPrefix(sp.Name, os.Getenv("BOT_SERVICE_PREFIX") + "port-") {
			port = sp.Port
			break
		}
	}

	path := networkingv1beta1.HTTPIngressPath{
		Path: "/",
		Backend: networkingv1beta1.IngressBackend{
			ServiceName: sn,
			ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: port},
		},
	}
	paths = append(paths, path)

	return
}

func getRules(sn, h string, ports []corev1.ServicePort) (rules []networkingv1beta1.IngressRule) {
	irv := networkingv1beta1.HTTPIngressRuleValue{Paths: getPaths(sn, ports)}

	rule := networkingv1beta1.IngressRule{
		Host: h,
		IngressRuleValue: networkingv1beta1.IngressRuleValue{HTTP: &irv},
	}
	rules = append(rules, rule)

	return
}
