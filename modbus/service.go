package modbus

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strconv"
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

	// ReadRegister performs an immediate Modbus read on behalf of the thing, publishes the
	// decoded value as a message of the thing and returns it. No client is created or looked up.
	ReadRegister(ctx context.Context, token string, req ReadRequest) (any, error)

	// WriteRegister performs an immediate Modbus write on behalf of the thing, reads the value back
	// and publishes it as a message of the thing. No client is created or looked up.
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

	// ErrUnsupportedWriteType indicates the field type cannot be written to a register.
	ErrUnsupportedWriteType = errors.New("unsupported write type")

	// ErrAddressOutOfRange indicates a field extends past the last Modbus address.
	ErrAddressOutOfRange = errors.New("field address out of range")

	// ErrDeviceUnavailable indicates the device is unreachable, timed out
	// or returned an invalid response.
	ErrDeviceUnavailable = errors.New("modbus device unavailable")

	// ErrDeviceRejected indicates the device returned a Modbus exception response.
	ErrDeviceRejected = errors.New("modbus device rejected request")

	errRateLimiter   = "failed to wait for rate limiter"
	errFormatPayload = "failed to format payload"
	errGetConnection = "failed to get connection"
	errEmptyResponse = "empty response payload"
)

// deviceError wraps err with the sentinel error matching the failure.
func deviceError(err error) error {
	var mbErr *gbmodbus.ModbusError
	if stderrors.As(err, &mbErr) {
		return errors.Wrap(ErrDeviceRejected, err)
	}

	return errors.Wrap(ErrDeviceUnavailable, err)
}

// isRequestError reports whether err is caused by the requested value rather than by the device.
func isRequestError(err error) bool {
	return errors.Contains(err, ErrInvalidWriteValue) ||
		errors.Contains(err, ErrValueOutOfRange) ||
		errors.Contains(err, ErrUnsupportedWriteType) ||
		errors.Contains(err, ErrAddressOutOfRange)
}

func New(things domain.ThingsClient, pub Publisher, clients ClientRepository, idp uuid.IDProvider, logger logger.Logger) Service {
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
		if err := validateFieldAddresses(clients[i].DataFields); err != nil {
			return []Client{}, err
		}
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
	if err := cs.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: req.ThingID, Action: domain.GroupEditor}); err != nil {
		return nil, errors.Wrap(errors.ErrAuthorization, err)
	}

	config, err := cs.things.GetConfigByThing(ctx, req.ThingID)
	if err != nil {
		return nil, err
	}

	field := DataField{
		Name:    messageKey(req.Name, req.Address),
		Unit:    req.Unit,
		Address: req.Address,
	}
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
	if err := validateFieldAddresses(fields); err != nil {
		return nil, err
	}

	handler, mu, err := cs.connect(ctx, req.IPAddress, req.Port, req.SlaveID)
	if err != nil {
		return nil, deviceError(err)
	}
	defer mu.Unlock()

	maxLen := getBlockMaxLen(req.FunctionCode)
	blocks := createBlocks(fields, maxLen)

	client := Client{
		IPAddress:    req.IPAddress,
		Port:         req.Port,
		SlaveID:      req.SlaveID,
		FunctionCode: req.FunctionCode,
		DataFields:   fields,
	}

	data, err := readData(gbmodbus.NewClient(handler), client, blocks)
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

	payload, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	if err := cs.publish(config, req.ThingID, payload); err != nil {
		return nil, err
	}

	// Register values are wrapped in an entry; coil values are plain booleans.
	if entry, ok := value.(map[string]any); ok {
		value = entry["value"]
	}

	return value, nil
}

// messageKey returns the key of a value published for an immediate read or write; without a name the address is used.
func messageKey(name string, address uint16) string {
	if name != "" {
		return name
	}

	return strconv.Itoa(int(address))
}

func (cs *clientsService) WriteRegister(ctx context.Context, token string, req WriteRequest) error {
	if err := cs.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: req.ThingID, Action: domain.GroupEditor}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	config, err := cs.things.GetConfigByThing(ctx, req.ThingID)
	if err != nil {
		return err
	}

	funcCode := WriteHoldingRegistersFunc
	if req.Type == BoolType {
		funcCode = WriteCoilsFunc
	}

	field := DataField{
		Name:      messageKey(req.Name, req.Address),
		Unit:      req.Unit,
		Type:      req.Type,
		ByteOrder: req.ByteOrder,
		Scale:     req.Scale,
		Address:   req.Address,
		Value:     req.Value,
	}
	fields := calcFieldLengths([]DataField{field})
	if err := validateFieldAddresses(fields); err != nil {
		return err
	}

	handler, mu, err := cs.connect(ctx, req.IPAddress, req.Port, req.SlaveID)
	if err != nil {
		return deviceError(err)
	}
	defer mu.Unlock()

	client := Client{FunctionCode: funcCode, DataFields: fields}
	blocks := createWriteBlocks(fields, getBlockMaxLen(funcCode))

	if err := writeData(gbmodbus.NewClient(handler), client, blocks); err != nil {
		if isRequestError(err) {
			return err
		}
		return deviceError(err)
	}

	// The written value is read back and published, as scheduled write clients do.
	readClient := client
	readClient.FunctionCode = readFunc(funcCode)

	data, err := readData(gbmodbus.NewClient(handler), readClient, blocks)
	if err != nil {
		return deviceError(err)
	}

	payload, err := formatPayload(data, readClient.DataFields, readClient.FunctionCode)
	if err != nil {
		return deviceError(err)
	}

	return cs.publish(config, req.ThingID, payload)
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

	handler, mu := cs.connPool.Get(key)

	mu.Lock()
	if err := handler.Connect(); err != nil {
		mu.Unlock()
		return nil, nil, fmt.Errorf("%s: %w", errGetConnection, err)
	}
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

	client.DataFields = calcFieldLengths(client.DataFields)
	if err := validateFieldAddresses(client.DataFields); err != nil {
		return err
	}

	if err = cs.clients.Update(ctx, client); err != nil {
		return err
	}

	cs.unscheduleTask(c)
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

	if err := cs.clients.Remove(ctx, ids...); err != nil {
		return err
	}

	for _, client := range cls {
		cs.unscheduleTask(client)
	}

	return nil
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
	cs.scheduler.RemoveCronEntry(c.ID)
	cs.scheduler.RemoveTimer(c.ID)
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

		data, err := readData(gbmodbus.NewClient(handler), readClient, blocks)
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

	pc := domain.PubConfigInfo{ThingID: thingID, ProfileConfig: config}
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
