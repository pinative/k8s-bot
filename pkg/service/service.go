package service

import (
	"errors"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"os"
	"reflect"
	"strings"
)

type Service struct {
	K8sClient kubernetes.Interface
	Name string
	Namespace string
}

func (s *Service) DeleteService(sif informers.SharedInformerFactory, l map[string]string) (err error) {
	svcLister := sif.Core().V1().Services().Lister()
	ns := s.Namespace
	ret, err := svcLister.Services(ns).List(labels.Set(l).AsSelector())
	if err != nil {
		log.Error().Err(err).Msgf("onDelete - Error to list services by labels %v from namespace %s", l, ns)
		return err
	}
	for _, svc := range ret {
		deletePolicy := metav1.DeletePropagationForeground
		err := s.K8sClient.CoreV1().Services(ns).Delete(svc.Name, &metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		})
		if err != nil {
			log.Error().Err(err).Msgf("onDelete - Error to delete the service %s in namespace %s", svc.Name, ns)
			return err
		}
	}

	return
}

func (s *Service) UpsertService(sfi informers.SharedInformerFactory, ol map[string]string, newDeploy *appsv1.Deployment, oldDeploy *appsv1.Deployment) (err error) {
	if len(ol) == 0 {
		return errors.New("invalid arguments, the labels should not be empty")
	}

	svcLister := sfi.Core().V1().Services().Lister()
	ns := s.Namespace
	services, err := svcLister.Services(ns).List(labels.Set(ol).AsSelector())
	if err != nil {
		log.Error().Err(err).Msgf("Error to list services by labels %v from namespace %s", ol, ns)
		return err
	}
	// As long as one or more available replicas alive
	//  then create a service for that deployment
	if len(services) == 0 && newDeploy.Status.AvailableReplicas > 0 {
		svc := newService(newDeploy)
		_, err = s.K8sClient.CoreV1().Services(newDeploy.GetNamespace()).Create(svc)
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			log.Error().
				Err(err).
				Str("namespace", svc.Namespace).
				Str("name", svc.Name).
				Send()
			return err
		}

	} else if oldDeploy.ResourceVersion != newDeploy.ResourceVersion && !reflect.DeepEqual(ol, newDeploy.GetLabels()) {
		for _, svc := range services {
			svc.Labels = newDeploy.Labels
			svc.Spec.Selector = newDeploy.GetLabels()
			_, err = s.K8sClient.CoreV1().Services(ns).Update(svc)
			if err != nil {
				log.Error().
					Err(err).
					Str("namespace", svc.Namespace).
					Str("name", svc.Name).
					Send()
				return err
			}
		}
	}

	return
}

func GetServicePort(sn string, ports []v1.ServicePort) int32 {
	for _, p := range ports {
		svcPrefix := os.Getenv("BOT_SERVICE_PREFIX")
		n := strings.TrimPrefix(sn, svcPrefix)
		pn := getServicePortName(svcPrefix, n)
		if pn == p.Name {
			return p.Port
		}
	}

	return 0
}

func newService(d *appsv1.Deployment) *v1.Service {
	c := getSpecificContainer("main", d.Spec.Template.Spec.Containers)
	port := getHttpContainerPort(c)
	svcPrefix := os.Getenv("BOT_SERVICE_PREFIX")

	aia := d.Annotations["pigo.network/allow-internet-access"]
	if aia == "" {
		aia = "false"
	}
	annots := map[string]string{"pigo.io/part-of":os.Getenv("ANNOT_PIGO_IO_PARTOF"), "pigo.network/allow-internet-access":aia}

	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: svcPrefix +  d.GetName(),
			Namespace: d.GetNamespace(),
			Labels: d.GetLabels(),
			Annotations: annots,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name: getServicePortName(svcPrefix, d.Name),
					Protocol: v1.ProtocolTCP,
					Port: port,
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: port},
				},
			},
			Type: v1.ServiceTypeClusterIP,
			Selector: d.GetLabels(),
		},
	}
}

func getServicePortName(sp, n string) string {
	return sp + "port-" + n
}

func getSpecificContainer(cn string, c []v1.Container) v1.Container {
	if len(c) == 1 {
		return c[0]
	}

	for _, v := range c {
		if cn == v.Name {
			return v
		}
	}

	return v1.Container{}
}

func getHttpContainerPort(c v1.Container) int32 {
	if len(c.Ports) == 1 {
		return c.Ports[0].ContainerPort
	}

	for _, p := range c.Ports {
		if p.Name == "http" {
			return p.ContainerPort
		}
	}

	return 0
}