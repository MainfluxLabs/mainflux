package modbus

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/protoutil"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	gbmodbus "github.com/goburrow/modbus"
	"golang.org/x/time/rate"
)

var AllowedOrders = map[string]string{
	"id":   "id",
	"name": "LOWER(name)",
}

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// All methods that accept a token parameter use it to identify and authorize
// the user performing the operation.
type Service interface {
	// CreateClients creates clients for certain thing identified by the thing ID.
	CreateClients(ctx context.Context, token, thingID string, Clients ...Client) ([]Client, error)

	// ListClientsByThing retrieves data about a subset of clients
	// related to a certain thing.
	ListClientsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (ClientsPage, error)

	// ListClientsByGroup retrieves data about a subset of clients
	// related to a certain group.
	ListClientsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (ClientsPage, error)

	// ViewClient retrieves data about a client identified with the provided ID.
	ViewClient(ctx context.Context, token, id string) (Client, error)

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
	RescheduleTasks(ctx context.Context, profileID string, config map[string]any) error

	// LoadAndScheduleTasks loads schedulers and starts them to execute requests based on client configuration.
	LoadAndScheduleTasks(ctx context.Context) error
}

type clientsService struct {
	things     protomfx.ThingsServiceClient
	clients    ClientRepository
	idProvider uuid.IDProvider
	publisher  messaging.Publisher
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

	maxRegs = 125  // 0x03 and 0x04
	maxBits = 2000 // 0x01 and 0x02
)

var _ Service = (*clientsService)(nil)
var (
	errRateLimiter    = "failed to wait for rate limiter"
	errFormatPayload  = "failed to format payload"
	errGetConnection  = "failed to get connection"
	errEmptyResponse  = "empty response payload"
	errNotEnoughBytes = "not enough bytes to read"
)

type Block struct {
	Start  uint16
	Length uint16
}

func New(things protomfx.ThingsServiceClient, pub messaging.Publisher, clients ClientRepository, idp uuid.IDProvider, logger logger.Logger) Service {
	return &clientsService{
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

func (cs *clientsService) CreateClients(ctx context.Context, token, thingID string, clients ...Client) ([]Client, error) {
	if _, err := cs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Editor}); err != nil {
		return nil, errors.Wrap(errors.ErrAuthorization, err)
	}

	grID, err := cs.things.GetGroupIDByThing(ctx, &protomfx.ThingID{Value: thingID})
	if err != nil {
		return []Client{}, err
	}
	groupID := grID.GetValue()

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

func (cs *clientsService) ListClientsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (ClientsPage, error) {
	if _, err := cs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Viewer}); err != nil {
		return ClientsPage{}, errors.Wrap(errors.ErrAuthorization, err)
	}

	page, err := cs.clients.RetrieveByThing(ctx, thingID, pm)
	if err != nil {
		return ClientsPage{}, err
	}

	return page, nil
}

func (cs *clientsService) ListClientsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (ClientsPage, error) {
	if _, err := cs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer}); err != nil {
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

	if _, err := cs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: client.ThingID, Action: things.Viewer}); err != nil {
		return Client{}, err
	}

	return client, nil
}

func (cs *clientsService) UpdateClient(ctx context.Context, token string, client Client) error {
	c, err := cs.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return err
	}

	if _, err := cs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: c.ThingID, Action: things.Editor}); err != nil {
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
	for _, id := range ids {
		client, err := cs.clients.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}
		if _, err := cs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: client.ThingID, Action: things.Editor}); err != nil {
			return err
		}

		cs.unscheduleTask(client)
	}

	return cs.clients.Remove(ctx, ids...)
}

func (cs *clientsService) RemoveClientsByThing(ctx context.Context, thingID string) error {
	page, err := cs.clients.RetrieveByThing(ctx, thingID, apiutil.PageMetadata{})
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
	page, err := cs.clients.RetrieveByGroup(ctx, groupID, apiutil.PageMetadata{})
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

func (cs *clientsService) RescheduleTasks(ctx context.Context, profileID string, config map[string]any) error {
	var clients []Client

	thingIDs, err := cs.things.GetThingIDsByProfile(ctx, &protomfx.ProfileID{Value: profileID})
	if err != nil {
		return err
	}

	for _, thingID := range thingIDs.GetIds() {
		page, err := cs.clients.RetrieveByThing(ctx, thingID, apiutil.PageMetadata{})
		if err != nil {
			return err
		}
		clients = append(clients, page.Clients...)
	}

	if len(clients) == 0 {
		return nil
	}

	cfg := protoutil.MapToProtoConfig(config)

	// stop existing tasks and start new tasks with updated config
	for _, d := range clients {
		cs.unscheduleTask(d)

		if err := cs.scheduleTask(d, cfg); err != nil {
			return err
		}
	}

	return nil
}

func (cs *clientsService) scheduleTasks(ctx context.Context, clients ...Client) error {
	for _, client := range clients {
		c, err := cs.things.GetConfigByThing(ctx, &protomfx.ThingID{Value: client.ThingID})
		if err != nil {
			return err
		}

		if err := cs.scheduleTask(client, c.GetConfig()); err != nil {
			return err
		}
	}

	return nil
}

func (cs *clientsService) scheduleTask(c Client, cfg *protomfx.Config) error {
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

func (cs *clientsService) createTask(client Client, config *protomfx.Config) func() {
	maxLen := getBlockMaxLen(client.FunctionCode)
	blocks := createBlocks(client.DataFields, maxLen)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		key := fmt.Sprintf("%s:%s", client.IPAddress, client.Port)
		limiter := cs.getLimiter(key)
		if err := limiter.Wait(ctx); err != nil {
			cs.logger.Error(fmt.Sprintf("%s: %s", errRateLimiter, err))
			return
		}

		handler, err := cs.connPool.Get(key)
		if err != nil {
			cs.logger.Error(fmt.Sprintf("%s: %s", errGetConnection, err))
			return
		}
		handler.SlaveId = client.SlaveID

		data, err := cs.readData(handler, client, blocks)
		if err != nil {
			cs.logger.Error(err.Error())
			return
		}
		if len(data) == 0 {
			cs.logger.Error(errEmptyResponse)
			return
		}

		formattedPayload, err := formatPayload(data, client.DataFields, client.FunctionCode)
		if err != nil {
			cs.logger.Error(fmt.Sprintf("%s: %s", errFormatPayload, err))
			return
		}

		if err := cs.publish(config, client.ThingID, formattedPayload); err != nil {
			cs.logger.Error(err.Error())
			return
		}
	}
}

func (cs *clientsService) readData(handler *gbmodbus.TCPClientHandler, client Client, blocks []Block) (map[string][]byte, error) {
	mc := gbmodbus.NewClient(handler)
	data := make(map[string][]byte)

	for _, block := range blocks {
		var (
			raw []byte
			err error
		)

		switch client.FunctionCode {
		case ReadCoilsFunc:
			raw, err = mc.ReadCoils(block.Start, block.Length)
		case ReadDiscreteInputsFunc:
			raw, err = mc.ReadDiscreteInputs(block.Start, block.Length)
		case ReadInputRegistersFunc:
			raw, err = mc.ReadInputRegisters(block.Start, block.Length)
		default:
			raw, err = mc.ReadHoldingRegisters(block.Start, block.Length)
		}
		if err != nil {
			return nil, err
		}

		// extract fields from block
		for _, field := range client.DataFields {
			if field.Address >= block.Start && field.Address < block.Start+block.Length {
				bytes, err := extractFieldBytes(raw, field, block, client.FunctionCode)
				if err != nil {
					return nil, err
				}
				data[field.Name] = bytes
			}
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
			return nil, fmt.Errorf("out of range coil %s", field.Name)
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
			return nil, fmt.Errorf("out of range for register %s", field.Name)
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
	default:
		maxLen = maxRegs
	}
	return
}

func formatPayload(data map[string][]byte, fields []DataField, funcCode string) ([]byte, error) {
	switch funcCode {
	case ReadCoilsFunc, ReadDiscreteInputsFunc:
		return formatCoilsPayload(data, fields)
	default:
		return formatRegistersPayload(data, fields)
	}
}

func formatRegistersPayload(data map[string][]byte, fields []DataField) ([]byte, error) {
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

	return json.Marshal(result)
}

func formatCoilsPayload(dataMap map[string][]byte, fields []DataField) ([]byte, error) {
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

	return json.Marshal(result)
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

func (cs *clientsService) publish(config *protomfx.Config, thingID string, payload []byte) error {
	msg := protomfx.Message{
		Protocol: modbusProtocol,
		Payload:  payload,
	}

	conn := &protomfx.PubConfigByKeyRes{PublisherID: thingID, ProfileConfig: config}
	if err := messaging.FormatMessage(conn, &msg); err != nil {
		return err
	}

	msg.Subject = nats.GetMessagesSubject(msg.Publisher, msg.Subtopic)
	if err := cs.publisher.Publish(msg); err != nil {
		return err
	}

	return nil
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
