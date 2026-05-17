package network

import (
	"encoding/json"
	"time"
)

type Interface struct {
	ID         int64     `json:"id"`
	NodeID     int64     `json:"node_id"`
	Name       string    `json:"name"`
	MACAddress string    `json:"mac_address,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	Netmask    string    `json:"netmask,omitempty"`
	Gateway    string    `json:"gateway,omitempty"`
	MTU        int       `json:"mtu"`
	Status     string    `json:"status"`
	IfaceType  string    `json:"iface_type"`
	BondMaster string    `json:"bond_master,omitempty"`
	Bridge     string    `json:"bridge,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Bridge struct {
	ID         int64           `json:"id"`
	NodeID     int64           `json:"node_id"`
	Name       string          `json:"name"`
	IPAddress  string          `json:"ip_address,omitempty"`
	Netmask    string          `json:"netmask,omitempty"`
	Members    json.RawMessage `json:"members"`
	STPEnabled bool            `json:"stp_enabled"`
	Status     string          `json:"status"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type Bond struct {
	ID        int64           `json:"id"`
	NodeID    int64           `json:"node_id"`
	Name      string          `json:"name"`
	Mode      string          `json:"mode"`
	IPAddress string          `json:"ip_address,omitempty"`
	Netmask   string          `json:"netmask,omitempty"`
	Members   json.RawMessage `json:"members"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type Route struct {
	ID          int64     `json:"id"`
	NodeID      int64     `json:"node_id"`
	Destination string    `json:"destination"`
	Netmask     string    `json:"netmask"`
	Gateway     string    `json:"gateway"`
	Interface   string    `json:"interface,omitempty"`
	Metric      int       `json:"metric"`
	CreatedAt   time.Time `json:"created_at"`
}

// Request types

type CreateInterfaceRequest struct {
	NodeID     int64  `json:"node_id"`
	Name       string `json:"name"`
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Netmask    string `json:"netmask"`
	Gateway    string `json:"gateway"`
	MTU        int    `json:"mtu"`
	IfaceType  string `json:"iface_type"`
	BondMaster string `json:"bond_master"`
	Bridge     string `json:"bridge"`
}

type UpdateInterfaceRequest struct {
	IPAddress *string `json:"ip_address"`
	Netmask   *string `json:"netmask"`
	Gateway   *string `json:"gateway"`
	MTU       *int    `json:"mtu"`
	Status    *string `json:"status"`
}

type CreateBridgeRequest struct {
	NodeID     int64           `json:"node_id"`
	Name       string          `json:"name"`
	IPAddress  string          `json:"ip_address"`
	Netmask    string          `json:"netmask"`
	Members    json.RawMessage `json:"members"`
	STPEnabled bool            `json:"stp_enabled"`
}

type UpdateBridgeRequest struct {
	IPAddress  *string          `json:"ip_address"`
	Netmask    *string          `json:"netmask"`
	Members    *json.RawMessage `json:"members"`
	STPEnabled *bool            `json:"stp_enabled"`
	Status     *string          `json:"status"`
}

type CreateBondRequest struct {
	NodeID    int64           `json:"node_id"`
	Name      string          `json:"name"`
	Mode      string          `json:"mode"`
	IPAddress string          `json:"ip_address"`
	Netmask   string          `json:"netmask"`
	Members   json.RawMessage `json:"members"`
}

type UpdateBondRequest struct {
	Mode      *string          `json:"mode"`
	IPAddress *string          `json:"ip_address"`
	Netmask   *string          `json:"netmask"`
	Members   *json.RawMessage `json:"members"`
	Status    *string          `json:"status"`
}

type CreateRouteRequest struct {
	NodeID      int64  `json:"node_id"`
	Destination string `json:"destination"`
	Netmask     string `json:"netmask"`
	Gateway     string `json:"gateway"`
	Interface   string `json:"interface"`
	Metric      int    `json:"metric"`
}

type UpdateRouteRequest struct {
	Destination *string `json:"destination"`
	Netmask     *string `json:"netmask"`
	Gateway     *string `json:"gateway"`
	Interface   *string `json:"interface"`
	Metric      *int    `json:"metric"`
}

type SlaveRequest struct {
	Slave string `json:"slave"`
}
