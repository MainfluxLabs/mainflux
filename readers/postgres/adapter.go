package postgres

import (
	"fmt"
	"github.com/MainfluxLabs/mainflux/readers"
)

const (
	jsonTable  = "json"
	jsonOrder  = "created"
	senmlTable = "messages"
	senmlOrder = "time"
)

type PageMetadataAdapter interface {
	GetOffset() uint64
	GetLimit() uint64
	GetSubtopic() string
	GetPublisher() string
	GetProtocol() string
	GetFrom() int64
	GetTo() int64
	GetAggInterval() string
	GetAggType() string
	GetAggField() string

	GetTable() string
	GetTimeColumn() string

	GetQueryParams() map[string]interface{}
	GetConditions() []string

	// For SenML specific fields
	GetName() string
	GetValue() float64
	GetComparator() string
	GetBoolValue() bool
	GetStringValue() string
	GetDataValue() string
}

type JSONAdapter struct {
	metadata readers.JSONMetadata
}

func NewJSONAdapter(metadata readers.JSONMetadata) PageMetadataAdapter {
	return &JSONAdapter{metadata: metadata}
}

func (j *JSONAdapter) GetOffset() uint64      { return j.metadata.Offset }
func (j *JSONAdapter) GetLimit() uint64       { return j.metadata.Limit }
func (j *JSONAdapter) GetSubtopic() string    { return j.metadata.Subtopic }
func (j *JSONAdapter) GetPublisher() string   { return j.metadata.Publisher }
func (j *JSONAdapter) GetProtocol() string    { return j.metadata.Protocol }
func (j *JSONAdapter) GetFrom() int64         { return j.metadata.From }
func (j *JSONAdapter) GetTo() int64           { return j.metadata.To }
func (j *JSONAdapter) GetAggInterval() string { return j.metadata.AggInterval }
func (j *JSONAdapter) GetAggType() string     { return j.metadata.AggType }
func (j *JSONAdapter) GetAggField() string    { return j.metadata.AggField }
func (j *JSONAdapter) GetTable() string       { return jsonTable }
func (j *JSONAdapter) GetTimeColumn() string  { return jsonOrder }

func (j *JSONAdapter) GetName() string        { return "" }
func (j *JSONAdapter) GetValue() float64      { return 0 }
func (j *JSONAdapter) GetComparator() string  { return "" }
func (j *JSONAdapter) GetBoolValue() bool     { return false }
func (j *JSONAdapter) GetStringValue() string { return "" }
func (j *JSONAdapter) GetDataValue() string   { return "" }

func (j *JSONAdapter) GetQueryParams() map[string]interface{} {
	return map[string]interface{}{
		"limit":     j.metadata.Limit,
		"offset":    j.metadata.Offset,
		"subtopic":  j.metadata.Subtopic,
		"publisher": j.metadata.Publisher,
		"protocol":  j.metadata.Protocol,
		"from":      j.metadata.From,
		"to":        j.metadata.To,
	}
}

func (j *JSONAdapter) GetConditions() []string {
	var conditions []string

	if j.metadata.Subtopic != "" {
		conditions = append(conditions, "subtopic = :subtopic")
	}
	if j.metadata.Publisher != "" {
		conditions = append(conditions, "publisher = :publisher")
	}
	if j.metadata.Protocol != "" {
		conditions = append(conditions, "protocol = :protocol")
	}
	if j.metadata.From != 0 {
		conditions = append(conditions, fmt.Sprintf("%s >= :from", j.GetTimeColumn()))
	}
	if j.metadata.To != 0 {
		conditions = append(conditions, fmt.Sprintf("%s <= :to", j.GetTimeColumn()))
	}

	return conditions
}

type SenMLAdapter struct {
	metadata readers.SenMLMetadata
}

func NewSenMLAdapter(metadata readers.SenMLMetadata) PageMetadataAdapter {
	return &SenMLAdapter{metadata: metadata}
}
func (s *SenMLAdapter) GetOffset() uint64      { return s.metadata.Offset }
func (s *SenMLAdapter) GetLimit() uint64       { return s.metadata.Limit }
func (s *SenMLAdapter) GetSubtopic() string    { return s.metadata.Subtopic }
func (s *SenMLAdapter) GetPublisher() string   { return s.metadata.Publisher }
func (s *SenMLAdapter) GetProtocol() string    { return s.metadata.Protocol }
func (s *SenMLAdapter) GetFrom() int64         { return s.metadata.From }
func (s *SenMLAdapter) GetTo() int64           { return s.metadata.To }
func (s *SenMLAdapter) GetAggInterval() string { return s.metadata.AggInterval }
func (s *SenMLAdapter) GetAggType() string     { return s.metadata.AggType }
func (s *SenMLAdapter) GetAggField() string    { return s.metadata.AggField }
func (s *SenMLAdapter) GetTable() string       { return senmlTable }
func (s *SenMLAdapter) GetTimeColumn() string  { return senmlOrder }

func (s *SenMLAdapter) GetName() string        { return s.metadata.Name }
func (s *SenMLAdapter) GetValue() float64      { return s.metadata.Value }
func (s *SenMLAdapter) GetComparator() string  { return s.metadata.Comparator }
func (s *SenMLAdapter) GetBoolValue() bool     { return s.metadata.BoolValue }
func (s *SenMLAdapter) GetStringValue() string { return s.metadata.StringValue }
func (s *SenMLAdapter) GetDataValue() string   { return s.metadata.DataValue }

func (s *SenMLAdapter) GetQueryParams() map[string]interface{} {
	return map[string]interface{}{
		"limit":        s.metadata.Limit,
		"offset":       s.metadata.Offset,
		"subtopic":     s.metadata.Subtopic,
		"publisher":    s.metadata.Publisher,
		"name":         s.metadata.Name,
		"protocol":     s.metadata.Protocol,
		"value":        s.metadata.Value,
		"bool_value":   s.metadata.BoolValue,
		"string_value": s.metadata.StringValue,
		"data_value":   s.metadata.DataValue,
		"from":         s.metadata.From,
		"to":           s.metadata.To,
	}
}

func (s *SenMLAdapter) GetConditions() []string {
	var conditions []string

	if s.metadata.Subtopic != "" {
		conditions = append(conditions, "subtopic = :subtopic")
	}
	if s.metadata.Publisher != "" {
		conditions = append(conditions, "publisher = :publisher")
	}
	if s.metadata.Protocol != "" {
		conditions = append(conditions, "protocol = :protocol")
	}
	if s.metadata.Name != "" {
		conditions = append(conditions, "name = :name")
	}
	if s.metadata.Value != 0 {
		comparator := s.parseComparator()
		conditions = append(conditions, fmt.Sprintf("value %s :value", comparator))
	}
	if s.metadata.BoolValue {
		conditions = append(conditions, "bool_value = :bool_value")
	}
	if s.metadata.StringValue != "" {
		conditions = append(conditions, "string_value = :string_value")
	}
	if s.metadata.DataValue != "" {
		conditions = append(conditions, "data_value = :data_value")
	}
	if s.metadata.From != 0 {
		conditions = append(conditions, fmt.Sprintf("%s >= :from", s.GetTimeColumn()))
	}
	if s.metadata.To != 0 {
		conditions = append(conditions, fmt.Sprintf("%s <= :to", s.GetTimeColumn()))
	}

	return conditions
}

func (s *SenMLAdapter) parseComparator() string {
	switch s.metadata.Comparator {
	case readers.EqualKey:
		return "="
	case readers.LowerThanKey:
		return "<"
	case readers.LowerThanEqualKey:
		return "<="
	case readers.GreaterThanKey:
		return ">"
	case readers.GreaterThanEqualKey:
		return ">="
	default:
		return "="
	}
}
