package generator

import (
	"log"

	wk "github.com/google/gnostic/cmd/protoc-gen-openapi/generator/wellknown"
	v3 "github.com/google/gnostic/openapiv3"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// schemaForMessage ...
func (r *OpenAPIv3Reflector) schemaForMessage(message protoreflect.MessageDescriptor) *v3.SchemaOrReference {
	typeName := r.fullMessageTypeName(message)
	switch typeName {

	case ".google.api.HttpBody":
		return wk.NewGoogleApiHttpBodySchema()

	case ".google.protobuf.Timestamp":
		return wk.NewGoogleProtobufTimestampSchema()

	case ".google.type.Date":
		return wk.NewGoogleTypeDateSchema()

	case ".google.type.DateTime":
		return wk.NewGoogleTypeDateTimeSchema()

	case ".google.protobuf.FieldMask":
		return wk.NewGoogleProtobufFieldMaskSchema()

	case ".google.protobuf.Struct":
		return wk.NewGoogleProtobufStructSchema()

	case ".google.protobuf.Empty":
		// Empty is closer to JSON undefined than null, so ignore this field
		return nil //&v3.SchemaOrReference{Oneof: &v3.SchemaOrReference_Schema{Schema: &v3.Schema{Type: "null"}}}

	default:

		var required []string
		definitionProperties := &v3.Properties{
			AdditionalProperties: make([]*v3.NamedSchemaOrReference, 0),
		}

		numFields := message.Fields().Len()
		for i := 0; i < numFields; i++ {
			fieldDescriptor := message.Fields().Get(i)
			schema := r.schemaForField(fieldDescriptor)

			definitionProperties.AdditionalProperties = append(
				definitionProperties.AdditionalProperties,
				&v3.NamedSchemaOrReference{
					Name:  r.formatFieldName(fieldDescriptor),
					Value: schema,
				},
			)
		}

		return &v3.SchemaOrReference{
			Oneof: &v3.SchemaOrReference_Schema{
				Schema: &v3.Schema{
					Type:        "object",
					Description: "",
					Properties:  definitionProperties,
					Required:    required,
				},
			}}
	}
}

// schemaForField ...
func (r *OpenAPIv3Reflector) schemaForField(field protoreflect.FieldDescriptor) *v3.SchemaOrReference {
	var kindSchema *v3.SchemaOrReference

	kind := field.Kind()

	switch kind {

	case protoreflect.MessageKind:
		if field.IsMap() {
			// This means the field is a map, for example:
			//   map<string, value_type> map_field = 1;
			//
			// The map ends up getting converted into something like this:
			//   message MapFieldEntry {
			//     string key = 1;
			//     value_type value = 2;
			//   }
			//
			//   repeated MapFieldEntry map_field = N;
			//
			// So we need to find the `value` field in the `MapFieldEntry` message and
			// then return a MapFieldEntry schema using the schema for the `value` field
			return wk.NewGoogleProtobufMapFieldEntrySchema(r.schemaOrReferenceForField(field.MapValue()))
		} else {
			kindSchema = r.schemaForMessage(field.Message())
		}

	case protoreflect.StringKind:
		kindSchema = wk.NewStringSchema()

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind,
		protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		kindSchema = wk.NewIntegerSchema(kind.String())

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind,
		protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		kindSchema = wk.NewStringSchema()

	case protoreflect.EnumKind:
		kindSchema = wk.NewEnumSchema(*&r.conf.EnumType, field)

	case protoreflect.BoolKind:
		kindSchema = wk.NewBooleanSchema()

	case protoreflect.FloatKind, protoreflect.DoubleKind:
		kindSchema = wk.NewNumberSchema(kind.String())

	case protoreflect.BytesKind:
		kindSchema = wk.NewBytesSchema()

	default:
		log.Printf("(TODO) Unsupported field type: %+v", r.fullMessageTypeName(field.Message()))
	}

	if field.IsList() {
		kindSchema = wk.NewListSchema(kindSchema)
	}

	return kindSchema
}
