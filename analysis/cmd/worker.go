package main

import (
	"context"
	"log"
	"os"

	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/gcppubsub"

	"github.com/ossf/package-analysis/analysis"
)

func messageLoop(ctx context.Context, sub *pubsub.Subscription, resultsBucket string) {
	for {
		msg, err := sub.Receive(ctx)
		if err != nil {
			// All subsequent receive calls will return the same error, so we bail out.
			log.Printf("error receiving message: %v", err)
			return
		}

		name := msg.Metadata["name"]
		if name == "" {
			log.Printf("name is empty")
			continue
		}

		ecosystem := msg.Metadata["ecosystem"]
		if ecosystem == "" {
			log.Printf("ecosystem is empty")
			continue
		}

		manager, ok := analysis.SupportedPkgManagers[ecosystem]
		if !ok {
			log.Printf("Unsupported pkg manager %s", manager)
			continue
		}
		log.Printf("Got request %s/%s", ecosystem, name)

		version := msg.Metadata["version"]
		if version == "" {
			version = manager.GetLatest(name)
		}
		log.Printf("Installing version %s", version)

		log.Printf("Got request %s/%s at version %s", ecosystem, name, version)
		info := analysis.Run(manager.Image, manager.CommandFmt(name, version))
		analysis.UploadResults(resultsBucket, ecosystem+"/"+name, ecosystem, name, version, info)
		msg.Ack()
	}
}

func main() {
	ctx := context.Background()
	subURL := os.Getenv("OSSMALWARE_WORKER_SUBSCRIPTION")
	resultsBucket := os.Getenv("OSSF_MALWARE_ANALYSIS_RESULTS")

	for {
		sub, err := pubsub.OpenSubscription(ctx, subURL)
		if err != nil {
			log.Panic(err)
		}

		messageLoop(ctx, sub, resultsBucket)
	}
}
