package controller

var (
	ExcludesNamespaceList = []string{ "kube-system", "ingress-nginx", "kube-public", "monitor" }
)

type BotController interface {
	Sync(stopCh <-chan struct{}) error
	Run(stopCh <-chan struct{})
	onAddFunc(obj interface{})
	onUpdateFunc(old, new interface{})
	onDeleteFunc(obj interface{})
}
