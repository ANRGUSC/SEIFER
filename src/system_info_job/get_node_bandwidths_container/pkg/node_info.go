package node_bandwidths

type BandwidthInfo struct {
	Start     string
	End       string
	Bandwidth float64
}

type NodeInfo struct {
	Bandwidths []*BandwidthInfo
	NodeMemory uint64
}
