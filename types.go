package descriptor

import (
	"reflect"
)

const (
	KindInvalid       = Kind(reflect.Invalid)
	KindBool          = Kind(reflect.Bool)
	KindInt           = Kind(reflect.Int)
	KindInt8          = Kind(reflect.Int8)
	KindInt16         = Kind(reflect.Int16)
	KindInt32         = Kind(reflect.Int32)
	KindInt64         = Kind(reflect.Int64)
	KindUint          = Kind(reflect.Uint)
	KindUint8         = Kind(reflect.Uint8)
	KindUint16        = Kind(reflect.Uint16)
	KindUint32        = Kind(reflect.Uint32)
	KindUint64        = Kind(reflect.Uint64)
	KindUintptr       = Kind(reflect.Uintptr)
	KindFloat32       = Kind(reflect.Float32)
	KindFloat64       = Kind(reflect.Float64)
	KindComplex64     = Kind(reflect.Complex64)
	KindComplex128    = Kind(reflect.Complex128)
	KindArray         = Kind(reflect.Array)
	KindChan          = Kind(reflect.Chan)
	KindFunc          = Kind(reflect.Func)
	KindInterface     = Kind(reflect.Interface)
	KindMap           = Kind(reflect.Map)
	KindPtr           = Kind(reflect.Ptr)
	KindSlice         = Kind(reflect.Slice)
	KindString        = Kind(reflect.String)
	KindStruct        = Kind(reflect.Struct)
	KindUnsafePointer = Kind(reflect.UnsafePointer)
)

type (
	Describable interface {
		Describe(data interface{}) (object interface{}, success, failure int)
		GetPrototype() interface{}
	}

	Type reflect.Type

	Descriptor struct {
		Type   Type
		Filler Filler
	}

	Filler interface {
		Fill(value reflect.Value, data interface{}) (success, failure int)
	}

	Fillers []Filler

	ObjectFiller struct {
		ObjectPath  ObjectPath
		ValueSource ValueSource
	}

	ObjectPath interface {
		FetchValue(root reflect.Value) (value reflect.Value, ok bool)
		FetchObject(root interface{}) (object interface{}, ok bool)
	}

	ValueSource interface {
		ExtractObject(data interface{}) (object interface{}, ok bool)
	}

	ValueSources []ValueSource

	ObjectAtPath struct {
		ObjectPath     ObjectPath
		AssignableKind AssignableKind
	}

	RootAsPath struct{}

	Path []interface{}

	Kind reflect.Kind

	AssignableKind interface {
		IsAssignable(i interface{}) bool
		AssignObject(i interface{}) (object interface{}, ok bool)
	}

	AssignableKinds []AssignableKind

	ConvertibleKind struct {
		Kind            Kind
		ConvertFunction func(original interface{}) (converted interface{}, ok bool)
	}

	AssignmentFunction func(i interface{}) (object interface{}, ok bool)

	DefaultValue struct {
		Value interface{}
	}
)

var Root = RootAsPath{}

func (d *Descriptor) Describe(data interface{}) (object interface{}, success, failure int) {
	if d == nil || d.Type == nil || d.Filler == nil {
		return nil, 0, 0x7fff
	}
	newValue := reflect.New(d.Type).Elem()
	success, failure = d.Filler.Fill(newValue, data)
	if !newValue.CanInterface() {
		return nil, 0, 0x7fff
	}
	object = newValue.Interface()
	return
}
func (d *Descriptor) GetPrototype() interface{} {
	if d == nil || d.Type == nil {
		return nil
	}
	e := reflect.New(d.Type).Elem()
	if !e.CanInterface() {
		return nil
	}
	return e.Interface()
}

func (f Fillers) Fill(value reflect.Value, data interface{}) (success, failure int) {
	success = 0
	failure = 0
	for _, filler := range f {
		if filler == nil {
			continue
		}
		s, f := filler.Fill(value, data)
		success += s
		failure += f
	}
	return
}

func (of ObjectFiller) Fill(value reflect.Value, data interface{}) (success, failure int) {
	objectPath := of.ObjectPath
	if objectPath == nil {
		objectPath = Root
	}
	fetchedValue, ok := objectPath.FetchValue(value)
	if !ok {
		return 0, 1
	}
	if !fetchedValue.CanSet() {
		return 0, 1
	}
	extractedObject, ok := of.ValueSource.ExtractObject(data)
	if !ok {
		return 0, 1
	}
	extractedValue := reflect.ValueOf(extractedObject)
	if !extractedValue.Type().AssignableTo(fetchedValue.Type()) {
		return 0, 1
	}
	fetchedValue.Set(extractedValue)
	return 1, 0
}

func (rap RootAsPath) FetchValue(root reflect.Value) (value reflect.Value, ok bool) {
	return root, true
}
func (rap RootAsPath) FetchObject(root interface{}) (object interface{}, ok bool) {
	return root, true
}

func (op Path) FetchValue(root reflect.Value) (value reflect.Value, ok bool) {
	v := root
	for _, index := range op {
		v, ok = valueAtIndex(reflect.ValueOf(index), v)
		if !ok {
			return
		}
	}
	return v, true
}
func (op Path) FetchObject(root interface{}) (object interface{}, ok bool) {
	v, ok := op.FetchValue(reflect.ValueOf(root))
	if !ok {
		return
	}
	if ok = v.CanInterface(); !ok {
		return
	}
	return v.Interface(), true
}

func (k Kind) IsAssignable(i interface{}) bool {
	return KindOf(i) == k
}
func (k Kind) AssignObject(i interface{}) (object interface{}, ok bool) {
	return i, true
}

func (ck ConvertibleKind) IsAssignable(i interface{}) bool {
	return KindOf(i) == ck.Kind
}
func (ck ConvertibleKind) AssignObject(i interface{}) (object interface{}, ok bool) {
	if ck.ConvertFunction == nil {
		return nil, false
	}
	return ck.ConvertFunction(i)
}

func (af AssignmentFunction) IsAssignable(interface{}) bool {
	return af != nil
}
func (af AssignmentFunction) AssignObject(i interface{}) (object interface{}, ok bool) {
	if af == nil {
		return nil, false
	}
	return af(i)
}

func (ak AssignableKinds) IsAssignable(interface{}) bool {
	return ak != nil && len(ak) > 0
}
func (ak AssignableKinds) AssignObject(i interface{}) (object interface{}, ok bool) {
	for _, assignableKind := range ak {
		if assignableKind == nil {
			continue
		}
		if !assignableKind.IsAssignable(i) {
			continue
		}
		object, ok = assignableKind.AssignObject(object)
		if ok {
			return
		}
	}
	return nil, false
}

func (oap ObjectAtPath) ExtractObject(data interface{}) (object interface{}, ok bool) {
	objectPath := oap.ObjectPath
	if objectPath == nil {
		objectPath = Root
	}
	ofd, ok := objectPath.FetchObject(data)
	if !ok {
		return
	}
	if ok = oap.AssignableKind != nil; !ok {
		return
	}
	if ok = oap.AssignableKind.IsAssignable(ofd); !ok {
		return
	}
	assignedObject, ok := oap.AssignableKind.AssignObject(ofd)
	if !ok {
		return
	}
	return assignedObject, true
}

func (dv DefaultValue) ExtractObject(interface{}) (object interface{}, ok bool) {
	return dv.Value, true
}

func (vs ValueSources) ExtractObject(data interface{}) (object interface{}, ok bool) {
	for _, valueSource := range vs {
		if valueSource == nil {
			continue
		}
		object, ok = valueSource.ExtractObject(data)
		if ok {
			return
		}
	}
	return nil, false
}

func TypeOfNew(ptr interface{}) Type {
	t := reflect.TypeOf(ptr)
	if t == nil || t.Kind() != reflect.Ptr {
		return nil
	}
	return t.Elem()
}

func PointerOf(i interface{}) interface{} {
	if i == nil {
		return nil
	}
	value := reflect.ValueOf(i)
	pointerValue := reflect.New(value.Type())
	pointerValue.Elem().Set(value)
	return pointerValue.Interface()
}

func KindOf(i interface{}) Kind {
	return kindOfValue(reflect.ValueOf(i))
}

func kindOfValue(v reflect.Value) Kind {
	return Kind(v.Kind())
}

func valueAtIndex(index, collection reflect.Value) (value reflect.Value, ok bool) {
	switch kindOfValue(collection) {
	case KindArray, KindSlice, KindString:
		switch indexKind := kindOfValue(index); indexKind {
		case KindInt, KindInt8, KindInt16, KindInt32, KindInt64,
			KindUint, KindUintptr, KindUint8, KindUint16, KindUint32, KindUint64:
			var i int
			switch indexKind {
			case KindInt, KindInt8, KindInt16, KindInt32, KindInt64:
				i = int(index.Int())
			case KindUint, KindUintptr, KindUint8, KindUint16, KindUint32, KindUint64:
				i = int(index.Uint())
			}
			length := collection.Len()
			if length < 1 || i < 0 || i >= length {
				return reflect.Value{}, false
			}
			return collection.Index(i), true
		default:
			return reflect.Value{}, false
		}
	case KindMap:
		if !index.Type().AssignableTo(collection.Type().Key()) {
			return reflect.Value{}, false
		}
		val := collection.MapIndex(index)
		if !val.IsValid() {
			return reflect.Value{}, false
		}
		return val, true
	case KindPtr:
		if collection.IsNil() {
			if !collection.CanSet() {
				return reflect.Value{}, false
			}
			collection.Set(reflect.New(collection.Type().Elem()))
		}
		return valueAtIndex(index, collection.Elem())
	case KindStruct:
		switch kindOfValue(index) {
		case KindString:
			val := collection.FieldByName(index.String())
			if !val.IsValid() {
				return reflect.Value{}, false
			}
			return val, true
		default:
			return reflect.Value{}, false
		}
	default:
		return reflect.Value{}, false
	}
}
