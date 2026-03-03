package modbus

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
)

const (
	modbusProtocol = "modbus"

	ReadCoilsFunc            = "ReadCoils"            // 0x01
	ReadDiscreteInputsFunc   = "ReadDiscreteInputs"   // 0x02
	ReadHoldingRegistersFunc = "ReadHoldingRegisters" // 0x03
	ReadInputRegistersFunc   = "ReadInputRegisters"   // 0x04

	ByteOrderABCD = "ABCD" // Big-endian (standard network order)
	ByteOrderDCBA = "DCBA" // Little-endian (x86, ARM in little-endian mode)
	ByteOrderCDAB = "CDAB" // Middle-endian (PDP-style)
	ByteOrderBADC = "BADC" // Byte-swapped (less common, but used in some contexts)
)

type Client struct {
	ID           string
	GroupID      string
	ThingID      string
	Name         string
	IPAddress    string
	Port         string
	SlaveID      uint8
	FunctionCode string
	Scheduler    cron.Scheduler
	Metadata     map[string]any
	DataFields   []DataField
}

type DataField struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Unit      string  `json:"unit"`
	Scale     float64 `json:"scale"`
	ByteOrder string  `json:"byte_order"`
	Address   uint16  `json:"address"`
	Length    uint16  `json:"length"`
}

type ClientsPage struct {
	apiutil.PageMetadata
	Clients []Client
}

type ClientRepository interface {
	// Save persists multiple Modbus clients.
	// Clients are saved using a transaction.
	// If one client fails, then none will be saved.
	// Successful operation is indicated by non-nil error response.
	Save(ctx context.Context, dls ...Client) ([]Client, error)

	// RetrieveByThing retrieves clients related to
	// a certain thing identified by a given ID.
	RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (ClientsPage, error)

	// RetrieveByGroup retrieves Modbus clients related to
	// a certain group identified by a given ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (ClientsPage, error)

	// RetrieveByID retrieves a client having the provided ID.
	RetrieveByID(ctx context.Context, id string) (Client, error)

	// RetrieveAll retrieves all clients.
	RetrieveAll(ctx context.Context) ([]Client, error)

	// Update performs an update to the existing client.
	// A non-nil error is returned to indicate operation failure.
	Update(ctx context.Context, w Client) error

	// Remove removes Modbus clients having the provided IDs.
	Remove(ctx context.Context, ids ...string) error

	// RemoveByThing removes Modbus clients related to
	// a certain thing identified by a given thing ID.
	RemoveByThing(ctx context.Context, thingID string) error

	// RemoveByGroup removes Modbus clients related to
	// a certain group identified by a given group ID.
	RemoveByGroup(ctx context.Context, groupID string) error
}
