package main

import (
	"context"

	nodebandwidths "github.com/Dat-Boi-Arjun/DEFER/system_init_job/get_node_bandwidths_container/pkg"
)

func main() {
	ctx := context.Background()
	nodebandwidths.RunBandwidthTasks(ctx)
}
