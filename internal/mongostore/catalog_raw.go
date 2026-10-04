package mongostore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// rawValueJSON encodes one stored BSON value as JSON exactly like the
// legacy Express encoder produced: field order preserved from the
// stored document, ObjectIds as hex strings, and dates as RFC 3339 timestamps.
// BSON types the legacy capture never covered
// (binary, decimal, regex, timestamps, code, DB pointers) render as
// MongoDB extended JSON so no valid stored document can fail encoding.
func rawValueJSON(value bson.RawValue) (json.RawMessage, error) {
	var out bytes.Buffer
	stream := &rawJSONEncoder{out: &out}
	if err := stream.value(value); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// rawJSONEncoder renders one stored BSON value as compact JSON.
type rawJSONEncoder struct {
	out *bytes.Buffer
}

// value renders one BSON value.
func (stream *rawJSONEncoder) value(value bson.RawValue) error {
	switch value.Type {
	case bson.TypeObjectID:
		stream.out.WriteString(strconv.Quote(value.ObjectID().Hex()))
	case bson.TypeDateTime:
		encoded, err := json.Marshal(value.Time().UTC())
		if err != nil {
			return err
		}
		stream.out.Write(encoded)
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
		bson.TypeJavaScript, bson.TypeCodeWithScope,
		bson.TypeDecimal128, bson.TypeTimestamp:
		encoded, err := extJSONValue(value)
		if err != nil {
			return err
		}
		stream.out.Write(encoded)
	default:
		return fmt.Errorf("unsupported BSON type %s", value.Type)
	}

	return nil
}

// extJSONWrapper positions one BSON value inside a document so the
// driver's extended JSON encoder accepts it; top-level values cannot
// hold every type.
type extJSONWrapper struct {
	Value bson.RawValue `bson:"v"`
}

// extJSONValue renders a BSON value as MongoDB extended JSON, the
// driver-supported conversion boundary for types without a pinned
// legacy shape.
func extJSONValue(value bson.RawValue) (json.RawMessage, error) {
	encoded, err := bson.MarshalExtJSON(
		extJSONWrapper{Value: value}, false, false,
	)
	if err != nil {
		return nil, fmt.Errorf("encode BSON value: %w", err)
	}

	const prefix = `{"v":`
	const suffix = `}`
	if !bytes.HasPrefix(encoded, []byte(prefix)) ||
		!bytes.HasSuffix(encoded, []byte(suffix)) {
		return nil, fmt.Errorf(
			"unexpected extended JSON encoding %s", encoded,
		)
	}

	return json.RawMessage(encoded[len(prefix) : len(encoded)-len(suffix)]), nil
}

// document renders an embedded document in stored field order.
func (stream *rawJSONEncoder) document(raw bson.Raw) error {
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
func (stream *rawJSONEncoder) array(raw bson.RawArray) error {
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
func (stream *rawJSONEncoder) double(value float64) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	stream.out.Write(encoded)

	return nil
}
