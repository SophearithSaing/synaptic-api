package mongostore

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// marshalValue builds a raw value from a one-field document and drops
// the wrapper.
func marshalValue(t *testing.T, value any) bson.RawValue {
	t.Helper()

	raw, err := bson.Marshal(bson.D{{Key: "v", Value: value}})
	if err != nil {
		t.Fatal(err)
	}

	return bson.Raw(raw).Lookup("v")
}

// TestRawValueJSONConversionBoundary pins the complete stored-value to
// JSON boundary: pinned legacy shapes for covered types and relaxed
// extended JSON for the rest, so no valid BSON value can break a
// response.
func TestRawValueJSONConversionBoundary(t *testing.T) {
	legacy := func(value any) func(t *testing.T) bson.RawValue {
		return func(t *testing.T) bson.RawValue {
			return marshalValue(t, value)
		}
	}

	cases := []struct {
		name  string
		build func(t *testing.T) bson.RawValue
		want  string
	}{
		{
			name:  "objectIdRendersHex",
			build: legacy(bson.NilObjectID),
			want:  `"000000000000000000000000"`,
		},
		{
			name:  "stringPassesThrough",
			build: legacy("hello"),
			want:  `"hello"`,
		},
		{
			name:  "boolean",
			build: legacy(true),
			want:  "true",
		},
		{
			name:  "null",
			build: legacy(nil),
			want:  "null",
		},
		{
			name:  "int32",
			build: legacy(int32(42)),
			want:  "42",
		},
		{
			name:  "int64StaysExact",
			build: legacy(int64(-9007199254740993)),
			want:  "-9007199254740993",
		},
		{
			name:  "double",
			build: legacy(float64(1.5)),
			want:  "1.5",
		},
		{
			name: "dateTimeUsesRFC3339", build: func(t *testing.T) bson.RawValue {
				return marshalValue(
					t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				)
			},
			want: `"2026-01-01T00:00:00Z"`,
		},
		{
			name: "binaryUsesExtendedJSON", build: legacy(
				bson.Binary{Subtype: 0x00, Data: []byte{1, 2, 3}},
			),
			want: `{"$binary":{"base64":"AQID","subType":"00"}}`,
		},
		{
			name: "decimalUsesExtendedJSON", build: legacy(
				bson.NewDecimal128(1, 2),
			),
			want: `{"$numberDecimal":"1.8446744073709551618E-6157"}`,
		},
		{
			name: "regexUsesExtendedJSON", build: legacy(
				bson.Regex{Pattern: "a", Options: "i"},
			),
			want: `{"$regularExpression":{"pattern":"a","options":"i"}}`,
		},
		{
			name: "timestampUsesExtendedJSON", build: legacy(
				bson.Timestamp{T: 42, I: 7},
			),
			want: `{"$timestamp":{"t":42,"i":7}}`,
		},
		{
			name: "documentPreservesFieldOrder", build: func(t *testing.T) bson.RawValue {
				return marshalValue(t, bson.D{
					{Key: "z", Value: int32(1)},
					{Key: "a", Value: int32(2)},
				})
			},
			want: `{"z":1,"a":2}`,
		},
		{
			name: "arrayPreservesOrder", build: func(t *testing.T) bson.RawValue {
				return marshalValue(t, bson.A{
					int32(1), "two", bson.D{{Key: "k", Value: int32(3)}},
				})
			},
			want: `[1,"two",{"k":3}]`,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := rawValueJSON(test.build(t))
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if string(encoded) != test.want {
				t.Fatalf("got %s, want %s", encoded, test.want)
			}
		})
	}
}
