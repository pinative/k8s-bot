package main

import (
	"context"
	"github.com/pinative/k8s-bot/observer"
	"github.com/pinative/k8s-bot/pkg/helper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/runtime"
	"os"
)

func main() {
	helper.LoadEnvVariables()

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	runtime.ErrorHandlers = []func(error){
		func(err error) { log.Warn().Err(err).Msg("[k8s]") },
	}

	o := observer.New(helper.GetClientset())

	var eg errgroup.Group
	eg.Go(func() error {
		return o.Run(context.TODO())
	})
	if err := eg.Wait(); err != nil {
		log.Fatal().Err(err).Send()
	}
}