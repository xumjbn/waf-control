package aggregation

type SystemResource struct {
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryPercent  float64 `json:"memory_percent"`
	MemoryTotal    int64   `json:"memory_total"`
	DiskPercent    float64 `json:"disk_percent"`
	DiskTotal      int64   `json:"disk_total"`
	NetConnections int64   `json:"net_connections"`
	NetworkIO      string  `json:"network_io"`
}

type NicInterface struct {
	Name      string       `json:"name"`
	State     string       `json:"state"`
	Addresses []NicAddress `json:"addresses,omitempty"`
}

type NicAddress struct {
	IP string `json:"ip"`
}

type NodeBlock struct {
	SystemResources SystemResource `json:"system_resources"`
	Interfaces      []NicInterface `json:"interfaces"`
	Datetime        string         `json:"datetime"`
}

type SystemMonitorAll struct {
	Manager   *NodeBlock           `json:"manager,omitempty"`
	Instances []map[string]NodeBlock `json:"instance,omitempty"`
}

type NicStateAll struct {
	Manager   []NicInterface             `json:"manager"`
	Instances map[string][]NicInterface  `json:"instances"`
}

type AttackRecord struct {
	Datetime    string     `json:"datetime"`
	SrcIP       string     `json:"src_ip"`
	DstIP       string     `json:"dst_ip"`
	Host        string     `json:"host"`
	Action      string     `json:"action"`
	Severity    string     `json:"severity"`
	SrcGeoCoord [2]float64 `json:"src_geo_coord"`
	DstGeoCoord [2]float64 `json:"dst_geo_coord"`
}

type AttackStats struct {
	StatisticInfo []AttackRecord `json:"statistic_info"`
}

type AttackSeverity struct {
	Critical int64 `json:"critical"`
	High     int64 `json:"high"`
	Medium   int64 `json:"medium"`
	Low      int64 `json:"low"`
	Info     int64 `json:"info"`
}

type AttackSourceTop struct {
	SrcIP string `json:"src_ip"`
	Count int64  `json:"count"`
}

type SiteStats struct {
	On   int64 `json:"on"`
	Off  int64 `json:"off"`
	Idle int64 `json:"idle"`
}

type RunningMode struct {
	RunningMode string `json:"running_mode"`
}

type FlowTopItem struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type InstanceBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type InstanceList struct {
	Instances []InstanceBrief `json:"instances"`
}

type ChangeTimeRequest struct {
	Time string `json:"time"`
}
