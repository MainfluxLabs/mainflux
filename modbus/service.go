package modbus

import (
	"context"
	"encoding/binary"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	gbmodbus "github.com/goburrow/modbus"
	"golang.org/x/time/rate"
)

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total     uint64 `json:"total,omitempty"`
	Offset    uint64 `json:"offset,omitempty"`
	Limit     uint64 `json:"limit,omitempty"`
	Order     string `json:"order,omitempty"`
	Dir       string `json:"dir,omitempty"`
	Name      string `json:"name,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
	Port      string `json:"port,omitempty"`
	SlaveID   int64  `json:"slave_id,omitempty"`
	Frequency string `json:"frequency,omitempty"`
}

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// All methods that accept a token parameter use it to identify and authorize
// the user performing the operation.
type Service interface {
	// CreateClients creates clients for certain thing identified by the thing ID.
	CreateClients(ctx context.Context, token, thingID string, Clients ...Client) ([]Client, error)

	// CreateClient creates a single client for the thing identified by the thing ID,
	// with data fields parsed from an uploaded file.
	CreateClient(ctx context.Context, token, thingID string, client Client) (Client, error)

	// ListClientsByThing retrieves data about a subset of clients
	// related to a certain thing.
	ListClientsByThing(ctx context.Context, token, thingID string, pm PageMetadata) (ClientsPage, error)

	// ListClientsByGroup retrieves data about a subset of clients
	// related to a certain group.
	ListClientsByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (ClientsPage, error)

	// ViewClient retrieves data about a client identified with the provided ID.
	ViewClient(ctx context.Context, token, id string) (Client, error)

	// ReadRegister performs an immediate Modbus read against the given connection and
	// returns the decoded value. No client is created or looked up; the token is only
	// checked for validity, not against any specific resource.
	ReadRegister(ctx context.Context, token string, req ReadRequest) (any, error)

	// WriteRegister performs an immediate Modbus write against the given connection.
	// No client is created or looked up; the token is only checked for validity, not
	// against any specific resource.
	WriteRegister(ctx context.Context, token string, req WriteRequest) error

	// UpdateClient updates client identified by the provided ID.
	UpdateClient(ctx context.Context, token string, client Client) error

	// RemoveClients removes clients identified with the provided IDs.
	RemoveClients(ctx context.Context, token string, id ...string) error

	// RemoveClientsByThing removes clients related to the specified thing,
	// identified by the provided thing ID.
	RemoveClientsByThing(ctx context.Context, thingID string) error

	// RemoveClientsByGroup removes clients related to the specified group,
	// identified by the provided group ID.
	RemoveClientsByGroup(ctx context.Context, groupID string) error

	// RescheduleTasks reschedules all tasks for things related to the specified profile ID.
	RescheduleTasks(ctx context.Context, profileID string, config *domain.ProfileConfig) error

	// LoadAndScheduleTasks loads schedulers and starts them to execute requests based on client configuration.
	LoadAndScheduleTasks(ctx context.Context) error
}

// Publisher specifies the minimal publishing capability the modbus service needs.
type Publisher interface {
	messaging.MessageDispatcher
}

type clientsService struct {
	things     domain.ThingsClient
	auth       domain.AuthClient
	clients    ClientRepository
	idProvider uuid.IDProvider
	publisher  Publisher
	logger     logger.Logger
	scheduler  *cron.ScheduleManager
	limiters   map[string]*rate.Limiter
	limiterMux sync.Mutex
	connPool   *modbusConnectionPool
}

const (
	BoolType    = "bool"
	Int16Type   = "int16"
	Int32Type   = "int32"
	Uint16Type  = "uint16"
	Uint32Type  = "uint32"
	Float32Type = "float32"
	StringType  = "string"

	// maxRegs is below the protocol limits because some devices and gateways reject full-size register requests.
	maxRegs      = 64   // 0x03, 0x04 and 0x10
	maxBits      = 2000 // 0x01 and 0x02
	maxWriteBits = 1968 // 0x0F
)

var _ Service = (*clientsService)(nil)
var (
	// ErrInvalidWriteValue indicates the write value does not match the register type.
	ErrInvalidWriteValue = errors.New("invalid write value")

	// ErrValueOutOfRange indicates the write value is out of range for the register type.
	ErrValueOutOfRange = errors.New("value out of range for type")

	// ErrDeviceUnavailable indicates the device is unreachable, timed out
	// or returned an invalid response.
	ErrDeviceUnavailable = errors.New("modbus device unavailable")

	// ErrDeviceRejected indicates the device returned a Modbus exception response.
	ErrDeviceRejected = errors.New("modbus device rejected request")

	errReadRegisters  = "failed to read registers"
	errWriteRegisters = "failed to write registers"
	errRateLimiter    = "failed to wait for rate limiter"
	errFormatPayload  = "failed to format payload"
	errGetConnection  = "failed to get connection"
	errEmptyResponse  = "empty response payload"
	errNotEnoughBytes = "not enough bytes to read"

	errInvalidWriteValue    = "invalid write value"
	errUnsupportedWriteType = "unsupported write type"
)

// deviceError wraps err with the sentinel error matching the failure.
func deviceError(err error) error {
	var mbErr *gbmodbus.ModbusError
	if stderrors.As(err, &mbErr) {
		return errors.Wrap(ErrDeviceRejected, err)
	}

	return errors.Wrap(ErrDeviceUnavailable, err)
}

type Block struct {
	Start  uint16
	Length uint16
}

func New(auth domain.AuthClient, things domain.ThingsClient, pub Publisher, clients ClientRepository, idp uuid.IDProvider, logger logger.Logger) Service {
	return &clientsService{
		auth:       auth,
		things:     things,
		publisher:  pub,
		clients:    clients,
		idProvider: idp,
		logger:     logger,
		scheduler:  cron.NewScheduleManager(),
		limiters:   make(map[string]*rate.Limiter),
		connPool:   newModbusConnectionPool(2*time.Minute, 30*time.Second),
	}
}

func (cs *clientsService) createClients(ctx context.Context, token, thingID string, clients ...Client) ([]Client, error) {
	if err := cs.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupEditor}); err != nil {
		return nil, errors.Wrap(errors.ErrAuthorization, err)
	}

	grID, err := cs.things.GetGroupIDByThing(ctx, thingID)
	if err != nil {
		return []Client{}, err
	}
	groupID := grID

	for i := range clients {
		clients[i].ThingID = thingID
		clients[i].GroupID = groupID

		id, err := cs.idProvider.ID()
		if err != nil {
			return []Client{}, err
		}
		clients[i].ID = id

		clients[i].DataFields = calcFieldLengths(clients[i].DataFields)
	}

	cls, err := cs.clients.Save(ctx, clients...)
	if err != nil {
		return []Client{}, err
	}

	if err := cs.scheduleTasks(ctx, cls...); err != nil {
		return nil, err
	}

	return cls, nil
}

func (cs *clientsService) CreateClients(ctx context.Context, token, thingID string, clients ...Client) ([]Client, error) {
	return cs.createClients(ctx, token, thingID, clients...)
}

func (cs *clientsService) CreateClient(ctx context.Context, token, thingID string, client Client) (Client, error) {
	cls, err := cs.createClients(ctx, token, thingID, client)
	if err != nil {
		return Client{}, err
	}

	return cls[0], nil
}

func (cs *clientsService) ListClientsByThing(ctx context.Context, token, thingID string, pm PageMetadata) (ClientsPage, error) {
	if err := cs.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupViewer}); err != nil {
		return ClientsPage{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	page, err := cs.clients.RetrieveByThing(ctx, thingID, pm)
	if err != nil {
		return ClientsPage{}, err
	}

	return page, nil
}

func (cs *clientsService) ListClientsByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (ClientsPage, error) {
	if err := cs.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupViewer}); err != nil {
		return ClientsPage{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	page, err := cs.clients.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return ClientsPage{}, err
	}

	return page, nil
}

func (cs *clientsService) ViewClient(ctx context.Context, token, id string) (Client, error) {
	client, err := cs.clients.RetrieveByID(ctx, id)
	if err != nil {
		return Client{}, err
	}

	if err := cs.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: client.ThingID, Action: domain.GroupViewer}); err != nil {
		return Client{}, err
	}

	return client, nil
}

func (cs *clientsService) ReadRegister(ctx context.Context, token string, req ReadRequest) (any, error) {
	if _, err := cs.auth.Identify(ctx, token); err != nil {
		return nil, err
	}

	handler, mu, err := cs.connect(ctx, req.IPAddress, req.Port, req.SlaveID)
	if err != nil {
		return nil, deviceError(err)
	}
	defer mu.Unlock()

	field := DataField{Address: req.Address}
	switch req.FunctionCode {
	case ReadCoilsFunc, ReadDiscreteInputsFunc:
		// Coils and discrete inputs are single bits and take no decoding options.
	default:
		field.Type = req.Type
		field.ByteOrder = req.ByteOrder
		field.Scale = req.Scale
		field.Length = req.Length
	}

	fields := calcFieldLengths([]DataField{field})
	maxLen := getBlockMaxLen(req.FunctionCode)
	blocks := createBlocks(fields, maxLen)

	client := Client{IPAddress: req.IPAddress, Port: req.Port, SlaveID: req.SlaveID, FunctionCode: req.FunctionCode, DataFields: fields}

	data, err := cs.readData(handler, client, blocks)
	if err != nil {
		return nil, deviceError(err)
	}

	values, err := buildFieldValues(data, fields, req.FunctionCode)
	if err != nil {
		return nil, deviceError(err)
	}

	value, ok := values[field.Name]
	if !ok {
		return nil, deviceError(errors.New(errEmptyResponse))
	}

	// Register values are wrapped in an entry; coil values are plain booleans.
	if entry, ok := value.(map[string]any); ok {
		value = entry["value"]
	}

	return value, nil
}

func (cs *clientsService) WriteRegister(ctx context.Context, token string, req WriteRequest) error {
	if _, err := cs.auth.Identify(ctx, token); err != nil {
		return err
	}

	handler, mu, err := cs.connect(ctx, req.IPAddress, req.Port, req.SlaveID)
	if err != nil {
		return deviceError(err)
	}
	defer mu.Unlock()

	funcCode := WriteHoldingRegistersFunc
	if req.Type == BoolType {
		funcCode = WriteCoilsFunc
	}

	fields := calcFieldLengths([]DataField{{Type: req.Type, ByteOrder: req.ByteOrder, Scale: req.Scale, Address: req.Address, Value: req.Value}})
	client := Client{FunctionCode: funcCode, DataFields: fields}
	blocks := createWriteBlocks(fields, getBlockMaxLen(funcCode))

	if err := writeData(gbmodbus.NewClient(handler), client, blocks); err != nil {
		if errors.Contains(err, ErrInvalidWriteValue) || errors.Contains(err, ErrValueOutOfRange) {
			return err
		}
		return deviceError(err)
	}

	return nil
}

// connect returns a rate-limited, connected handler for ipAddress:port, ready for
// immediate I/O, along with its mutex already locked. Callers must defer mu.Unlock()
// after their I/O completes, since the connection may be shared by requests
// targeting different slave IDs. Callers are responsible for their own authorization.
func (cs *clientsService) connect(ctx context.Context, ipAddress, port string, slaveID uint8) (*gbmodbus.TCPClientHandler, *sync.Mutex, error) {
	key := fmt.Sprintf("%s:%s", ipAddress, port)

	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	limiter := cs.getLimiter(key)
	if err := limiter.Wait(waitCtx); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errRateLimiter, err)
	}

	handler, mu, err := cs.connPool.Get(key)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", errGetConnection, err)
	}

	mu.Lock()
	handler.SlaveId = slaveID

	return handler, mu, nil
}

func (cs *clientsService) UpdateClient(ctx context.Context, token string, client Client) error {
	c, err := cs.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return err
	}

	if err := cs.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: c.ThingID, Action: domain.GroupEditor}); err != nil {
		return err
	}

	cs.unscheduleTask(c)

	client.DataFields = calcFieldLengths(client.DataFields)

	if err = cs.clients.Update(ctx, client); err != nil {
		return err
	}
	client.ThingID = c.ThingID

	return cs.scheduleTasks(ctx, client)
}

func (cs *clientsService) RemoveClients(ctx context.Context, token string, ids ...string) error {
	cls := make([]Client, 0, len(ids))
	thingIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		client, err := cs.clients.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}

		cls = append(cls, client)
		thingIDs = append(thingIDs, client.ThingID)
	}

	if err := cs.things.CanUserAccessThings(ctx, domain.UserAccessThingsReq{Token: token, IDs: thingIDs, Action: domain.GroupEditor}); err != nil {
		return err
	}

	for _, client := range cls {
		cs.unscheduleTask(client)
	}

	return cs.clients.Remove(ctx, ids...)
}

func (cs *clientsService) RemoveClientsByThing(ctx context.Context, thingID string) error {
	page, err := cs.clients.RetrieveByThing(ctx, thingID, PageMetadata{})
	if err != nil {
		return err
	}

	if len(page.Clients) == 0 {
		return nil
	}

	for _, c := range page.Clients {
		cs.unscheduleTask(c)
	}

	return cs.clients.RemoveByThing(ctx, thingID)
}

func (cs *clientsService) RemoveClientsByGroup(ctx context.Context, groupID string) error {
	page, err := cs.clients.RetrieveByGroup(ctx, groupID, PageMetadata{})
	if err != nil {
		return err
	}

	if len(page.Clients) == 0 {
		return nil
	}

	for _, c := range page.Clients {
		cs.unscheduleTask(c)
	}

	return cs.clients.RemoveByGroup(ctx, groupID)
}

func (cs *clientsService) RescheduleTasks(ctx context.Context, profileID string, config *domain.ProfileConfig) error {
	var clients []Client

	thingIDs, err := cs.things.GetThingIDsByProfile(ctx, profileID)
	if err != nil {
		return err
	}

	for _, thingID := range thingIDs {
		page, err := cs.clients.RetrieveByThing(ctx, thingID, PageMetadata{})
		if err != nil {
			return err
		}
		clients = append(clients, page.Clients...)
	}

	if len(clients) == 0 {
		return nil
	}

	// stop existing tasks and start new tasks with updated config
	for _, d := range clients {
		cs.unscheduleTask(d)

		if err := cs.scheduleTask(d, config); err != nil {
			return err
		}
	}

	return nil
}

func (cs *clientsService) scheduleTasks(ctx context.Context, clients ...Client) error {
	for _, client := range clients {
		cfg, err := cs.things.GetConfigByThing(ctx, client.ThingID)
		if err != nil {
			return err
		}

		if err := cs.scheduleTask(client, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (cs *clientsService) scheduleTask(c Client, cfg *domain.ProfileConfig) error {
	task := cs.createTask(c, cfg)

	if c.Scheduler.Frequency != cron.OnceFreq {
		return cs.scheduler.ScheduleRepeatingTask(task, c.Scheduler, c.ID)
	}

	return cs.scheduler.ScheduleOneTimeTask(task, c.Scheduler, c.ID)
}

func (cs *clientsService) unscheduleTask(c Client) {
	if c.Scheduler.Frequency != cron.OnceFreq {
		cs.scheduler.RemoveCronEntry(c.ID, c.Scheduler.TimeZone)
	}

	if t, ok := cs.scheduler.TimerByID[c.ID]; ok {
		t.Stop()
		delete(cs.scheduler.TimerByID, c.ID)
	}
}

func (cs *clientsService) createTask(client Client, config *domain.ProfileConfig) func() {
	maxLen := getBlockMaxLen(client.FunctionCode)
	write := isWriteFunc(client.FunctionCode)

	var blocks []Block
	if write {
		blocks = createWriteBlocks(client.DataFields, maxLen)
	} else {
		blocks = createBlocks(client.DataFields, maxLen)
	}

	// Write clients read the written blocks back, so the published values reflect the device state.
	readClient := client
	readClient.FunctionCode = readFunc(client.FunctionCode)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		logPrefix := fmt.Sprintf("client %s (%s) %s:%s", client.ID, client.Name, client.IPAddress, client.Port)

		handler, mu, err := cs.connect(ctx, client.IPAddress, client.Port, client.SlaveID)
		if err != nil {
			cs.logger.Error(fmt.Sprintf("%s: %s", logPrefix, err))
			return
		}
		defer mu.Unlock()

		if write {
			if err := writeData(gbmodbus.NewClient(handler), client, blocks); err != nil {
				cs.logger.Error(fmt.Sprintf("%s: %s", logPrefix, err))
				return
			}

			cs.logger.Info(fmt.Sprintf("%s: wrote %d fields", logPrefix, len(client.DataFields)))
		}

		data, err := cs.readData(handler, readClient, blocks)
		if err != nil {
			cs.logger.Error(fmt.Sprintf("%s: %s", logPrefix, err))
			return
		}
		if len(data) == 0 {
			cs.logger.Error(fmt.Sprintf("%s: %s", logPrefix, errEmptyResponse))
			return
		}

		formattedPayload, err := formatPayload(data, readClient.DataFields, readClient.FunctionCode)
		if err != nil {
			cs.logger.Error(fmt.Sprintf("%s: %s: %s", logPrefix, errFormatPayload, err))
			return
		}

		if err := cs.publish(config, client.ThingID, formattedPayload); err != nil {
			cs.logger.Error(fmt.Sprintf("%s: %s", logPrefix, err))
			return
		}

		cs.logger.Info(fmt.Sprintf("%s: published %d fields", logPrefix, len(data)))
	}
}

func isWriteFunc(funcCode string) bool {
	switch funcCode {
	case WriteCoilsFunc, WriteHoldingRegistersFunc:
		return true
	default:
		return false
	}
}

// readFunc returns the read function code for the register table that a write function code targets.
func readFunc(funcCode string) string {
	switch funcCode {
	case WriteCoilsFunc:
		return ReadCoilsFunc
	case WriteHoldingRegistersFunc:
		return ReadHoldingRegistersFunc
	default:
		return funcCode
	}
}

// createWriteBlocks groups adjacent fields only, because a write request spanning a gap
// overwrites the addresses between fields.
func createWriteBlocks(fields []DataField, maxLen int) []Block {
	if len(fields) == 0 {
		return nil
	}

	slices.SortFunc(fields, func(a, b DataField) int {
		return int(a.Address) - int(b.Address)
	})

	var blocks []Block
	block := Block{Start: fields[0].Address, Length: fields[0].Length}

	for _, field := range fields[1:] {
		if field.Address == block.Start+block.Length && int(block.Length+field.Length) <= maxLen {
			block.Length += field.Length
			continue
		}

		blocks = append(blocks, block)
		block = Block{Start: field.Address, Length: field.Length}
	}

	return append(blocks, block)
}

func writeData(mc gbmodbus.Client, client Client, blocks []Block) error {
	for _, block := range blocks {
		fields := blockFields(client.DataFields, block)

		// Encoding errors are returned unwrapped to preserve the ErrInvalidWriteValue and ErrValueOutOfRange sentinels.
		raw, err := encodeBlock(client.FunctionCode, block, fields)
		if err != nil {
			return err
		}

		if err := writeBlock(mc, client.FunctionCode, block, raw); err != nil {
			names := make([]string, 0, len(fields))
			for _, field := range fields {
				names = append(names, fmt.Sprintf("%s@%d", field.Name, field.Address))
			}
			return fmt.Errorf("%s: write %s starting at %d, length %d, fields [%s]: %w", errWriteRegisters, client.FunctionCode, block.Start, block.Length, strings.Join(names, ", "), err)
		}
	}

	return nil
}

// encodeBlock converts the field values of a block into its write request payload.
func encodeBlock(funcCode string, block Block, fields []DataField) ([]byte, error) {
	switch funcCode {
	case WriteCoilsFunc:
		// Modbus packs coils LSB-first.
		bits := make([]byte, (block.Length+7)/8)
		for _, field := range fields {
			v, ok := field.Value.(bool)
			if !ok {
				return nil, errors.Wrap(ErrInvalidWriteValue, fmt.Errorf("expected bool value for field %s", field.Name))
			}
			if v {
				offset := int(field.Address - block.Start)
				byteIndex := offset / 8
				bitIndex := uint(offset % 8)
				bits[byteIndex] |= 1 << bitIndex
			}
		}
		return bits, nil
	default:
		raw := make([]byte, int(block.Length)*2)
		for _, field := range fields {
			b, err := encodeRegisterField(field)
			if err != nil {
				return nil, err
			}
			copy(raw[int(field.Address-block.Start)*2:], b)
		}
		return raw, nil
	}
}

// writeBlock uses function codes 0x05 and 0x06 for single-element blocks,
// because not all devices implement 0x0F and 0x10.
func writeBlock(mc gbmodbus.Client, funcCode string, block Block, raw []byte) error {
	switch funcCode {
	case WriteCoilsFunc:
		if block.Length == 1 {
			coilValue := uint16(0x0000)
			if raw[0] != 0 {
				coilValue = 0xFF00
			}

			_, err := mc.WriteSingleCoil(block.Start, coilValue)
			return err
		}

		_, err := mc.WriteMultipleCoils(block.Start, block.Length, raw)
		return err
	default:
		if block.Length == 1 {
			_, err := mc.WriteSingleRegister(block.Start, binary.BigEndian.Uint16(raw))
			return err
		}

		_, err := mc.WriteMultipleRegisters(block.Start, block.Length, raw)
		return err
	}
}

// encodeRegisterField converts the field value into raw register bytes in the field byte order.
func encodeRegisterField(field DataField) ([]byte, error) {
	var raw []byte
	switch field.Type {
	case BoolType:
		v, ok := field.Value.(bool)
		if !ok {
			return nil, errors.Wrap(ErrInvalidWriteValue, fmt.Errorf("expected bool value for field %s", field.Name))
		}

		raw = []byte{0x00, 0x00}
		if v {
			raw[1] = 0x01
		}
	default:
		v, ok := field.Value.(float64)
		if !ok {
			return nil, errors.Wrap(ErrInvalidWriteValue, fmt.Errorf("expected numeric value for field %s", field.Name))
		}

		var err error
		raw, err = encodeNumericField(v, field.Type, field.Scale)
		if err != nil {
			return nil, err
		}
	}

	return reorderBytes(raw, field.ByteOrder), nil
}

// ValidateWriteValue checks that value, after scale is applied, fits the numeric register type.
func ValidateWriteValue(value float64, typ string, scale float64) error {
	_, err := encodeNumericField(value, typ, scale)
	return err
}

// encodeNumericField converts an engineering-unit value back into raw big-endian
// device bytes, undoing the scale applied by readNumericField.
func encodeNumericField(value float64, typ string, scale float64) ([]byte, error) {
	if scale != 0 {
		value /= scale
	}

	switch typ {
	case Int16Type:
		rounded := math.Round(value)
		if rounded < math.MinInt16 || rounded > math.MaxInt16 {
			return nil, errors.Wrap(ErrValueOutOfRange, fmt.Errorf("%v out of range for %s", value, typ))
		}
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(int16(rounded)))
		return b, nil
	case Uint16Type:
		rounded := math.Round(value)
		if rounded < 0 || rounded > math.MaxUint16 {
			return nil, errors.Wrap(ErrValueOutOfRange, fmt.Errorf("%v out of range for %s", value, typ))
		}
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(rounded))
		return b, nil
	case Int32Type:
		rounded := math.Round(value)
		if rounded < math.MinInt32 || rounded > math.MaxInt32 {
			return nil, errors.Wrap(ErrValueOutOfRange, fmt.Errorf("%v out of range for %s", value, typ))
		}
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(int32(rounded)))
		return b, nil
	case Uint32Type:
		rounded := math.Round(value)
		if rounded < 0 || rounded > math.MaxUint32 {
			return nil, errors.Wrap(ErrValueOutOfRange, fmt.Errorf("%v out of range for %s", value, typ))
		}
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(rounded))
		return b, nil
	case Float32Type:
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, math.Float32bits(float32(value)))
		return b, nil
	default:
		return nil, fmt.Errorf("%s: %s", errUnsupportedWriteType, typ)
	}
}

func readBlock(mc gbmodbus.Client, funcCode string, start, length uint16) ([]byte, error) {
	switch funcCode {
	case ReadCoilsFunc:
		return mc.ReadCoils(start, length)
	case ReadDiscreteInputsFunc:
		return mc.ReadDiscreteInputs(start, length)
	case ReadInputRegistersFunc:
		return mc.ReadInputRegisters(start, length)
	default:
		return mc.ReadHoldingRegisters(start, length)
	}
}

// blockFields returns the client's data fields that fall inside the given block.
func blockFields(fields []DataField, block Block) []DataField {
	res := make([]DataField, 0, len(fields))
	for _, field := range fields {
		if field.Address >= block.Start && field.Address < block.Start+block.Length {
			res = append(res, field)
		}
	}
	return res
}

func (cs *clientsService) readData(handler *gbmodbus.TCPClientHandler, client Client, blocks []Block) (map[string][]byte, error) {
	mc := gbmodbus.NewClient(handler)
	data := make(map[string][]byte)

	for _, block := range blocks {
		fields := blockFields(client.DataFields, block)

		raw, err := readBlock(mc, client.FunctionCode, block.Start, block.Length)
		if err != nil {
			names := make([]string, 0, len(fields))
			for _, field := range fields {
				names = append(names, fmt.Sprintf("%s@%d", field.Name, field.Address))
			}
			return nil, fmt.Errorf("%s: read %s starting at %d, length %d, fields [%s]: %w", errReadRegisters, client.FunctionCode, block.Start, block.Length, strings.Join(names, ", "), err)
		}

		for _, field := range fields {
			bytes, err := extractFieldBytes(raw, field, block, client.FunctionCode)
			if err != nil {
				return nil, fmt.Errorf("field %s@%d: %w", field.Name, field.Address, err)
			}
			data[field.Name] = bytes
		}
	}
	return data, nil
}

func extractFieldBytes(raw []byte, field DataField, block Block, funcCode string) ([]byte, error) {
	switch funcCode {
	case ReadCoilsFunc, ReadDiscreteInputsFunc:
		// coils/discrete inputs: single bits (1 bit = 1 value)
		// calculate bit offset relative to block start
		bitOffset := int(field.Address - block.Start)

		// determine byte and bit inside byte that holds value
		byteIndex := bitOffset / 8
		bitIndex := uint(bitOffset % 8)

		if byteIndex >= len(raw) {
			return nil, fmt.Errorf("field %s at address %d: out of range for coil block starting at %d, length %d", field.Name, field.Address, block.Start, block.Length)
		}

		// extract bit: mask byte and check if bit is set
		bit := (raw[byteIndex] & (1 << bitIndex)) != 0

		// convert bit (bool) to single byte (0x01 = ON, 0x00 = OFF)
		if bit {
			return []byte{0x01}, nil
		}
		return []byte{0x00}, nil
	default:
		// registers: each is 2 bytes (16-bit)
		// convert register offset to byte offset
		startByte := int(field.Address-block.Start) * 2
		lenByte := int(field.Length) * 2

		if startByte+lenByte > len(raw) {
			return nil, fmt.Errorf("field %s at address %d, length %d: out of range for register block starting at %d, length %d", field.Name, field.Address, field.Length, block.Start, block.Length)
		}

		return raw[startByte : startByte+lenByte], nil
	}
}

func createBlocks(fields []DataField, maxLen int) []Block {
	if len(fields) == 0 {
		return nil
	}

	slices.SortFunc(fields, func(a, b DataField) int {
		return int(a.Address) - int(b.Address)
	})

	var blocks []Block
	blockStart := fields[0].Address
	blockEnd := fields[0].Address + fields[0].Length

	for _, field := range fields[1:] {
		// distance from start of block to end of current field
		distance := int(field.Address + field.Length - 1 - blockStart)

		if distance < maxLen {
			if field.Address+field.Length > blockEnd {
				blockEnd = field.Address + field.Length
			}
			continue
		}

		// close current block
		blocks = append(blocks, Block{
			Start:  blockStart,
			Length: blockEnd - blockStart,
		})

		// start new block
		blockStart = field.Address
		blockEnd = field.Address + field.Length
	}

	// add last block
	blocks = append(blocks, Block{
		Start:  blockStart,
		Length: blockEnd - blockStart,
	})

	return blocks
}

func getBlockMaxLen(funcCode string) (maxLen int) {
	switch funcCode {
	case ReadCoilsFunc, ReadDiscreteInputsFunc:
		maxLen = maxBits
	case WriteCoilsFunc:
		maxLen = maxWriteBits
	default:
		maxLen = maxRegs
	}
	return
}

func formatPayload(data map[string][]byte, fields []DataField, funcCode string) ([]byte, error) {
	result, err := buildFieldValues(data, fields, funcCode)
	if err != nil {
		return nil, err
	}

	return json.Marshal(result)
}

func buildFieldValues(data map[string][]byte, fields []DataField, funcCode string) (map[string]any, error) {
	switch funcCode {
	case ReadCoilsFunc, ReadDiscreteInputsFunc:
		return buildCoilValues(data, fields)
	default:
		return buildRegisterValues(data, fields)
	}
}

func buildRegisterValues(data map[string][]byte, fields []DataField) (map[string]any, error) {
	result := make(map[string]any)

	for _, f := range fields {
		raw, ok := data[f.Name]
		if !ok {
			continue
		}

		raw = reorderBytes(raw, f.ByteOrder)

		var (
			value any
			err   error
		)

		switch f.Type {
		case BoolType:
			if len(raw) < 2 {
				return nil, fmt.Errorf("%s: expected 2 bytes for bool, got %d", errNotEnoughBytes, len(raw))
			}
			bv := binary.BigEndian.Uint16(raw)
			value = bv == 1
		case Int16Type:
			value, err = readNumericField[int16](raw, f.Scale)
		case Uint16Type:
			value, err = readNumericField[uint16](raw, f.Scale)
		case Int32Type:
			value, err = readNumericField[int32](raw, f.Scale)
		case Uint32Type:
			value, err = readNumericField[uint32](raw, f.Scale)
		case Float32Type:
			value, err = readNumericField[float32](raw, f.Scale)
		case StringType:
			// ASCII and UTF-8 values
			str := string(raw)
			value = strings.TrimRight(str, "\x00")
		}
		if err != nil {
			return nil, err
		}
		result[f.Name] = createEntry(value, f.Unit)
	}

	return result, nil
}

func buildCoilValues(dataMap map[string][]byte, fields []DataField) (map[string]any, error) {
	result := make(map[string]any)

	for _, f := range fields {
		raw := dataMap[f.Name]
		if len(raw) == 0 {
			continue
		}
		// Modbus packs coils LSB-first: the addressed coil is always in bit 0 of raw[0].
		bit := (raw[0] & 0x01) != 0

		result[f.Name] = bit
	}

	return result, nil
}

func calcFieldLengths(fields []DataField) []DataField {
	for i := range fields {
		switch fields[i].Type {
		case Int32Type, Uint32Type, Float32Type:
			fields[i].Length = 2
		case StringType:
			continue
		default:
			fields[i].Length = 1
		}
	}

	return fields
}

func createEntry(value any, unit string) map[string]any {
	entry := map[string]any{"value": value}
	if unit != "" {
		entry["unit"] = unit
	}
	return entry
}

func readNumericField[T int16 | uint16 | int32 | uint32 | float32](data []byte, scale float64) (any, error) {
	var value T
	switch any(value).(type) {
	case int16:
		if len(data) < 2 {
			return nil, fmt.Errorf("%s: expected 2 bytes for int16, got %d", errNotEnoughBytes, len(data))
		}
		value = T(int16(binary.BigEndian.Uint16(data)))
	case uint16:
		if len(data) < 2 {
			return nil, fmt.Errorf("%s: expected 2 bytes for uint16, got %d", errNotEnoughBytes, len(data))
		}
		value = T(binary.BigEndian.Uint16(data))
	case int32:
		if len(data) < 4 {
			return nil, fmt.Errorf("%s: expected 4 bytes for int32, got %d", errNotEnoughBytes, len(data))
		}
		value = T(int32(binary.BigEndian.Uint32(data)))
	case uint32:
		if len(data) < 4 {
			return nil, fmt.Errorf("%s: expected 4 bytes for uint32, got %d", errNotEnoughBytes, len(data))
		}
		value = T(binary.BigEndian.Uint32(data))
	case float32:
		if len(data) < 4 {
			return nil, fmt.Errorf("%s: expected 4 bytes for float32, got %d", errNotEnoughBytes, len(data))
		}
		bits := binary.BigEndian.Uint32(data)
		value = T(math.Float32frombits(bits))
	}

	if scale != 0 {
		return float64(value) * scale, nil
	}
	return value, nil
}

func reorderBytes(data []byte, order string) []byte {
	bytes := append([]byte(nil), data...)

	switch order {
	case ByteOrderABCD:
		return bytes
	case ByteOrderDCBA:
		slices.Reverse(bytes)
	case ByteOrderCDAB:
		if len(bytes) == 4 {
			bytes = []byte{bytes[2], bytes[3], bytes[0], bytes[1]}
		}
	case ByteOrderBADC:
		if len(bytes) == 4 {
			bytes = []byte{bytes[1], bytes[0], bytes[3], bytes[2]}
		}
	}
	return bytes
}

func (cs *clientsService) LoadAndScheduleTasks(ctx context.Context) error {
	var clients []Client
	cls, err := cs.clients.RetrieveAll(ctx)
	if err != nil {
		return err
	}

	for _, c := range cls {
		if c.Scheduler.Frequency == cron.OnceFreq {
			scheduledDateTime, err := cron.ParseTime(cron.DateTimeLayout, c.Scheduler.DateTime, c.Scheduler.TimeZone)
			if err != nil {
				return err
			}

			now := time.Now().In(scheduledDateTime.Location())
			if scheduledDateTime.After(now) {
				clients = append(clients, c)
			}
			continue
		}
		clients = append(clients, c)
	}

	if err := cs.scheduleTasks(ctx, clients...); err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		cs.scheduler.Stop()
		cs.connPool.Close()
	}()

	return nil
}

func (cs *clientsService) publish(config *domain.ProfileConfig, thingID string, payload []byte) error {
	msg := protomfx.Message{
		Protocol: modbusProtocol,
		Payload:  payload,
	}

	pc := domain.PubConfigInfo{PublisherID: thingID, ProfileConfig: config}
	if err := messaging.FormatMessage(pc, &msg); err != nil {
		return err
	}

	return cs.publisher.Dispatch(msg, pc.ProfileConfig)
}

func (cs *clientsService) getLimiter(key string) *rate.Limiter {
	cs.limiterMux.Lock()
	defer cs.limiterMux.Unlock()

	limiter, exists := cs.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(1, 1)
		cs.limiters[key] = limiter
	}
	return limiter
}
