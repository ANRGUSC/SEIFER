package main

import (
	"context"

	deploypods "github.com/Dat-Boi-Arjun/DEFER/dispatcher_pod/config_step/deploy_container/pkg"
)

func main() {
	ctx := context.Background()
	deploypods.DeployInferencePods(ctx)
}
