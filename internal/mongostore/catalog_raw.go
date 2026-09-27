package mongostore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/SophearithSaing/synaptic-api/internal/catalog"
)

// ToRawJSON encodes BSON values as JSON exactly like the legacy
// Express encoder produced: field order preserved from the stored
// document, ObjectIds as hex strings, dates as JS ISO-8601 with
// millisecond precision.
func ToRawJSON(values ...bson.RawValue) (json.RawMessage, error) {
	if len(values) == 1 {
		return rawValueJSON(values[0])
	}

	var out bytes.Buffer
	stream := &rawJSONStreamer{out: &out}
	if err := stream.write(values...); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// rawJSONStreamer renders ordered BSON values as compact JSON.
type rawJSONStreamer struct {
	out *bytes.Buffer
}

// write renders each BSON value.
func (stream *rawJSONStreamer) write(values ...bson.RawValue) error {
	for _, value := range values {
		if err := stream.value(value); err != nil {
			return err
		}
	}

	return nil
}

// value renders one BSON value.
func (stream *rawJSONStreamer) value(value bson.RawValue) error {
	switch value.Type {
	case bson.TypeObjectID:
		stream.out.WriteString(strconv.Quote(value.ObjectID().Hex()))
	case bson.TypeDateTime:
		stream.out.WriteString(strconv.Quote(
			catalog.ISO8601(value.Time()),
		))
	case bson.TypeString, bson.TypeSymbol:
		encoded, err := json.Marshal(value.StringValue())
		if err != nil {
			return err
		}
		stream.out.Write(encoded)
	case bson.TypeEmbeddedDocument:
		return stream.document(value.Document())
	case bson.TypeArray:
		return stream.array(value.Array())
	case bson.TypeBoolean:
		if value.Boolean() {
			stream.out.WriteString("true")
		} else {
			stream.out.WriteString("false")
		}
	case bson.TypeNull, bson.TypeUndefined, bson.TypeMinKey,
		bson.TypeMaxKey:
		stream.out.WriteString("null")
	case bson.TypeInt32:
		stream.out.WriteString(strconv.FormatInt(int64(value.Int32()), 10))
	case bson.TypeInt64:
		stream.out.WriteString(strconv.FormatInt(value.Int64(), 10))
	case bson.TypeDouble:
		return stream.double(value.Double())
	case bson.TypeBinary, bson.TypeRegex, bson.TypeDBPointer,
		bson.TypeCodeWithScope, bson.TypeDecimal128, bson.TypeTimestamp:
		return fmt.Errorf("unsupported BSON type %s", value.Type)
	default:
		return fmt.Errorf("unsupported BSON type %s", value.Type)
	}

	return nil
}

// rawValueJSON encodes one stored value.
func rawValueJSON(value bson.RawValue) (json.RawMessage, error) {
	var out bytes.Buffer
	stream := &rawJSONStreamer{out: &out}
	if err := stream.value(value); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// document renders an embedded document in stored field order.
func (stream *rawJSONStreamer) document(raw bson.Raw) error {
	stream.out.WriteByte('{')

	first := true
	elements, visitErr := raw.Elements()
	if visitErr != nil {
		return visitErr
	}
	for _, element := range elements {
		if !first {
			stream.out.WriteByte(',')
		}
		first = false

		key := strings.ReplaceAll(strconv.Quote(element.Key()), "\\u0000", "")
		stream.out.WriteString(key)
		stream.out.WriteByte(':')

		if err := stream.value(element.Value()); err != nil {
			return err
		}
	}

	stream.out.WriteByte('}')

	return nil
}

// array renders a BSON array in stored order.
func (stream *rawJSONStreamer) array(raw bson.RawArray) error {
	stream.out.WriteByte('[')

	values, visitErr := raw.Values()
	if visitErr != nil {
		return visitErr
	}
	for position, value := range values {
		if position != 0 {
			stream.out.WriteByte(',')
		}

		if err := stream.value(value); err != nil {
			return err
		}
	}

	stream.out.WriteByte(']')

	return nil
}

// double renders a float using the shortest faithful representation.
func (stream *rawJSONStreamer) double(value float64) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	stream.out.Write(encoded)

	return nil
}
