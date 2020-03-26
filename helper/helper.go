package helper

import (
	"encoding/hex"
	"encoding/json"
	"github.com/rs/xid"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GetClientset() *kubernetes.Clientset {
	if clientset, err := kubernetes.NewForConfig(getKubernetesConfig()); err != nil {
		log.Fatal().Err(err).Send()
		panic(err)
	} else {
		return clientset
	}
}

func GetRandomValue(n int32) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		log.Error().Err(err).Send()
	}

	return hex.EncodeToString(b)
}

// AreNamespaceInWhiteList verifies if the provided label list
// is in the provided whitelist and returns true, otherwise false.
func AreNamespaceInWhiteList(namespace string, whitelist []string) bool {
	for _, ns := range whitelist {
		if ns == namespace {
			return true
		}
	}
	return false
}

func GetPublicDns() string {
	d := os.Getenv("PUBLIC_DNS_DOMAIN")
	if strings.HasPrefix(".", d) {
		return generateDNSPrefix() + d
	}

	return generateDNSPrefix() + "." + d
}

func FormatJson(v interface{}) (formatted []byte) {
	formatted, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Error().
			Err(err).
			Send()
	}

	return
}

func generateDNSPrefix() string {
	abc := GetRandomValue(6)
	dnsPrefix := xid.New().String() + abc
	return dnsPrefix
}

func getDNSPrefix(dns string) string {
	idx := strings.Index(dns, ".")
	return dns[0:idx]
}

func getKubernetesConfig() *rest.Config {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(homeDir(), ".kube", "config"))
	}
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get kubernetes configuration")
	}
	return config
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}