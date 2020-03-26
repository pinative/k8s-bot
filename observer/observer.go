package observer

import (
	"context"
	"fmt"
	botcntlr "github.com/pinative/k8s-bot/controller"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"os"
	"strconv"
	"sync"
	"time"
)

// A Observer observes for resources in the kubernetes cluster
type Observer struct {
	client   kubernetes.Interface
}

// New creates a new Observer.
func New(client kubernetes.Interface) *Observer {
	return &Observer{
		client:   client,
	}
}

// Run runs the watcher.
func (w *Observer) Run(ctx context.Context) error {
	rd := os.Getenv("RESYNC_DURATION_IN_SECONDS")
	if rd == "" {
		rd = "0"
	}
	i, err := strconv.Atoi(rd)
	if err != nil {
		log.Error().Msg("Error to read the environment variable RESYNC_DURATION")
		return err
	}

	factory := informers.NewSharedInformerFactoryWithOptions(w.client, time.Duration(i) * time.Second)
	factory.Core().V1().Services().Lister()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ingCntlr := botcntlr.NewIngressController(factory)
		defer runtime.HandleCrash()

		err = ingCntlr.Sync(ctx.Done())
		if err != nil {
			log.Fatal().Err(err).Send()
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync pod"))
		}

		select {
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		stopCh := make(chan struct{})
		defer close(stopCh)
		defer runtime.HandleCrash()

		svcCntlr := botcntlr.NewServiceController(factory)
		err := svcCntlr.Sync(stopCh)
		if err != nil {
			log.Fatal().Err(err).Send()
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync service"))
		}

		select {
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		deplCntlr := botcntlr.NewDeploymentController(factory)
		waitCh := make(chan struct{})
		defer close(waitCh)
		defer runtime.HandleCrash()

		err = deplCntlr.Sync(waitCh)
		if err != nil {
			log.Fatal().Err(err).Send()
			runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync pod"))
		}

		select {
		case <-waitCh:
			deplCntlr.Run(ctx.Done())
		}
	}()

	wg.Wait()
	return nil
}
