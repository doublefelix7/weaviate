//                           _       _
// __      _____  __ ___   ___  __ _| |_ ___
// \ \ /\ / / _ \/ _` \ \ / / |/ _` | __/ _ \
//  \ V  V /  __/ (_| |\ V /| | (_| | ||  __/
//   \_/\_/ \___|\__,_| \_/ |_|\__,_|\__\___|
//
//  Copyright © 2016 - 2023 Weaviate B.V. All rights reserved.
//
//  CONTACT: hello@weaviate.io
//

package schema

import (
	"errors"
	"fmt"
	"unicode"
)

type DataType string

const (
	// DataTypeCRef The data type is a cross-reference, it is starting with a capital letter
	DataTypeCRef DataType = "cref"
	// DataTypeText The data type is a value of type string
	DataTypeText DataType = "text"
	// DataTypeInt The data type is a value of type int
	DataTypeInt DataType = "int"
	// DataTypeNumber The data type is a value of type number/float
	DataTypeNumber DataType = "number"
	// DataTypeBoolean The data type is a value of type boolean
	DataTypeBoolean DataType = "boolean"
	// DataTypeDate The data type is a value of type date
	DataTypeDate DataType = "date"
	// DataTypeGeoCoordinates is used to represent geo coordinates, i.e. latitude
	// and longitude pairs of locations on earth
	DataTypeGeoCoordinates DataType = "geoCoordinates"
	// DataTypePhoneNumber represents a parsed/to-be-parsed phone number
	DataTypePhoneNumber DataType = "phoneNumber"
	// DataTypeBlob represents a base64 encoded data
	DataTypeBlob DataType = "blob"
	// DataTypeTextArray The data type is a value of type string array
	DataTypeTextArray DataType = "text[]"
	// DataTypeIntArray The data type is a value of type int array
	DataTypeIntArray DataType = "int[]"
	// DataTypeNumberArray The data type is a value of type number/float array
	DataTypeNumberArray DataType = "number[]"
	// DataTypeBooleanArray The data type is a value of type boolean array
	DataTypeBooleanArray DataType = "boolean[]"
	// DataTypeDateArray The data type is a value of type date array
	DataTypeDateArray DataType = "date[]"
	// DataTypeUUID is a native UUID data type. It is stored in it's raw byte
	// representation and therefore takes up less space than storing a UUID as a
	// string
	DataTypeUUID DataType = "uuid"
	// DataTypeUUIDArray is the array version of DataTypeUUID
	DataTypeUUIDArray DataType = "uuid[]"

	// deprecated as of v1.19, replaced by DataTypeText + relevant tokenization setting
	// DataTypeString The data type is a value of type string
	DataTypeString DataType = "string"
	// deprecated as of v1.19, replaced by DataTypeTextArray + relevant tokenization setting
	// DataTypeArrayString The data type is a value of type string array
	DataTypeStringArray DataType = "string[]"
)

func (dt DataType) String() string {
	return string(dt)
}

func (dt DataType) PropString() []string {
	return []string{dt.String()}
}

var PrimitiveDataTypes []DataType = []DataType{
	DataTypeText, DataTypeInt, DataTypeNumber, DataTypeBoolean, DataTypeDate,
	DataTypeGeoCoordinates, DataTypePhoneNumber, DataTypeBlob, DataTypeTextArray,
	DataTypeIntArray, DataTypeNumberArray, DataTypeBooleanArray, DataTypeDateArray,
	DataTypeUUID, DataTypeUUIDArray,
}

var DeprecatedPrimitiveDataTypes []DataType = []DataType{
	// deprecated as of v1.19
	DataTypeString, DataTypeStringArray,
}

type PropertyKind int

const (
	PropertyKindPrimitive PropertyKind = 1
	PropertyKindRef       PropertyKind = 2
)

type PropertyDataType interface {
	Kind() PropertyKind
	IsPrimitive() bool
	AsPrimitive() DataType
	IsReference() bool
	Classes() []ClassName
	ContainsClass(name ClassName) bool
}

type propertyDataType struct {
	kind          PropertyKind
	primitiveType DataType
	classes       []ClassName
}

// IsPropertyLength returns if a string is a filters for property length. They have the form len(*PROPNAME*)
func IsPropertyLength(propName string, offset int) (string, bool) {
	isPropLengthFilter := len(propName) > 4+offset && propName[offset:offset+4] == "len(" && propName[len(propName)-1:] == ")"

	if isPropLengthFilter {
		return propName[offset+4 : len(propName)-1], isPropLengthFilter
	}
	return "", false
}

func IsArrayType(dt DataType) (DataType, bool) {
	switch dt {
	case DataTypeStringArray:
		return DataTypeString, true
	case DataTypeTextArray:
		return DataTypeText, true
	case DataTypeNumberArray:
		return DataTypeNumber, true
	case DataTypeIntArray:
		return DataTypeInt, true
	case DataTypeBooleanArray:
		return DataTypeBoolean, true
	case DataTypeDateArray:
		return DataTypeDate, true
	case DataTypeUUIDArray:
		return DataTypeUUID, true

	default:
		return "", false
	}
}

func (p *propertyDataType) Kind() PropertyKind {
	return p.kind
}

func (p *propertyDataType) IsPrimitive() bool {
	return p.kind == PropertyKindPrimitive
}

func (p *propertyDataType) AsPrimitive() DataType {
	if p.kind != PropertyKindPrimitive {
		panic("not primitive type")
	}

	return p.primitiveType
}

func (p *propertyDataType) IsReference() bool {
	return p.kind == PropertyKindRef
}

func (p *propertyDataType) Classes() []ClassName {
	if p.kind != PropertyKindRef {
		panic("not MultipleRef type")
	}

	return p.classes
}

func (p *propertyDataType) ContainsClass(needle ClassName) bool {
	if p.kind != PropertyKindRef {
		panic("not MultipleRef type")
	}

	for _, class := range p.classes {
		if class == needle {
			return true
		}
	}

	return false
}

// Based on the schema, return a valid description of the defined datatype
//
// Note that this function will error if referenced classes do not exist. If
// you don't want such validation, use [Schema.FindPropertyDataTypeRelaxedRefs]
// instead and set relax to true
func (s *Schema) FindPropertyDataType(dataType []string) (PropertyDataType, error) {
	return s.FindPropertyDataTypeWithRefs(dataType, false, "")
}

// Based on the schema, return a valid description of the defined datatype
// If relaxCrossRefValidation is set, there is no check if the referenced class
// exists in the schema. This can be helpful in scenarios, such as restoring
// from a backup where we have no guarantee over the order of class creation.
// If belongingToClass is set and equal to referenced class, check whether class
// exists in the schema is skipped. This is done to allow creating class schema with
// properties referencing to itself. Previously such properties had to be created separately
// only after creation of class schema
func (s *Schema) FindPropertyDataTypeWithRefs(
	dataType []string, relaxCrossRefValidation bool, beloningToClass ClassName,
) (PropertyDataType, error) {
	if len(dataType) < 1 {
		return nil, errors.New("dataType must have at least one element")
	}
	if len(dataType) == 1 {
		for _, dt := range append(PrimitiveDataTypes, DeprecatedPrimitiveDataTypes...) {
			if dataType[0] == dt.String() {
				return &propertyDataType{
					kind:          PropertyKindPrimitive,
					primitiveType: dt,
				}, nil
			}
		}
		if len(dataType[0]) == 0 {
			return nil, fmt.Errorf("dataType cannot be an empty string")
		}
		firstLetter := rune(dataType[0][0])
		if unicode.IsLower(firstLetter) {
			return nil, fmt.Errorf("Unknown primitive data type '%s'", dataType[0])
		}
	}
	/* implies len(dataType) > 1, or first element is a class already */
	var classes []ClassName

	for _, someDataType := range dataType {
		className, err := ValidateClassName(someDataType)
		if err != nil {
			return nil, err
		}

		if beloningToClass != className && !relaxCrossRefValidation {
			if s.FindClassByName(className) == nil {
				return nil, ErrRefToNonexistentClass
			}
		}

		classes = append(classes, className)
	}

	return &propertyDataType{
		kind:    PropertyKindRef,
		classes: classes,
	}, nil
}

func AsPrimitive(dataType []string) (DataType, bool) {
	if (len(dataType)) == 1 {
		for _, dt := range append(PrimitiveDataTypes, DeprecatedPrimitiveDataTypes...) {
			if dataType[0] == dt.String() {
				return dt, true
			}
		}
		if len(dataType[0]) == 0 {
			return "", true
		}

		return "", unicode.IsLower(rune(dataType[0][0]))
	}
	return "", false
}
