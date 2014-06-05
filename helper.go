package g

/*
#include <stdlib.h>
#include <glib-object.h>
#cgo pkg-config: gobject-2.0

static inline GType gvalue_get_type(GValue *v) {
	return G_VALUE_TYPE(v);
}

static inline GType gtype_get_fundamental(GType t) {
	return G_TYPE_FUNDAMENTAL(t);
}

extern void closureMarshal(GClosure*, GValue*, guint, GValue*, gpointer, gpointer);

GClosure* new_closure(void *data) {
	GClosure *closure = g_closure_new_simple(sizeof(GClosure), NULL);
	g_closure_set_meta_marshal(closure, data, (GClosureMarshal)(closureMarshal));
	return closure;
}

extern void callIdleCb(void*);

static gboolean idleCb(void *cb) {
	callIdleCb(cb);
	return FALSE;
}

static inline GValue* gvalue_new() {
	return (GValue*)g_slice_alloc0(sizeof(GValue));
}

gboolean is_object(void *o) {
	return G_IS_OBJECT(o);
}

*/
import "C"

import (
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

// signal connect

var refHolder []interface{}
var refHolderLock sync.Mutex

var gconnect = GConnect
var gconnectAfter = GConnectAfter

func GConnect(obj interface{}, signal string, cb interface{}) uint64 {
	return _connect(obj, signal, cb, C.FALSE)
}

func GConnectAfter(obj interface{}, signal string, cb interface{}) uint64 {
	return _connect(obj, signal, cb, C.TRUE)
}

func _connect(obj interface{}, signal string, cb interface{}, after C.gboolean) uint64 {
	cbp := &cb
	refHolderLock.Lock()
	refHolder = append(refHolder, cbp) //FIXME deref
	refHolderLock.Unlock()
	closure := C.new_closure(unsafe.Pointer(cbp))
	cSignal := (*C.gchar)(unsafe.Pointer(C.CString(signal)))
	defer C.free(unsafe.Pointer(cSignal))
	id := C.g_signal_connect_closure(C.gpointer(unsafe.Pointer(reflect.ValueOf(obj).Pointer())),
		cSignal, closure, after)
	return uint64(id)
}

func fromGValue(v *C.GValue) (ret interface{}) {
	valueType := C.gvalue_get_type(v)
	fundamentalType := C.gtype_get_fundamental(valueType)
	switch fundamentalType {
	case C.G_TYPE_OBJECT:
		ret = unsafe.Pointer(C.g_value_get_object(v))
	case C.G_TYPE_STRING:
		ret = fromGStr(C.g_value_get_string(v))
	case C.G_TYPE_UINT:
		ret = int(C.g_value_get_uint(v))
	case C.G_TYPE_BOXED:
		ret = unsafe.Pointer(C.g_value_get_boxed(v))
	case C.G_TYPE_BOOLEAN:
		ret = C.g_value_get_boolean(v) == C.gboolean(1)
	case C.G_TYPE_ENUM:
		ret = int(C.g_value_get_enum(v))
	default:
		panic(fmt.Sprintf("from type %s %T", fromGStr(C.g_type_name(fundamentalType)), v))
	}
	return
}

// string

func fromGStr(s *C.gchar) string {
	return C.GoString((*C.char)(unsafe.Pointer(s)))
}

var _gstrs = make(map[string]*C.gchar)

func toGStr(s string) *C.gchar {
	if gstr, ok := _gstrs[s]; ok {
		return gstr
	}
	gstr := (*C.gchar)(unsafe.Pointer(C.CString(s)))
	_gstrs[s] = gstr
	return gstr
}

func ObjSet(o interface{}, name string, value interface{}) {
	obj := (*C.GObject)(unsafe.Pointer(reflect.ValueOf(o).Pointer()))
	C.g_object_set_property(obj, toGStr(name), toGValue(value))
}

func toGValue(v interface{}) *C.GValue {
	value := C.gvalue_new()
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		C.g_value_init(value, C.G_TYPE_STRING)
		cStr := C.CString(v.(string))
		defer C.free(unsafe.Pointer(cStr))
		C.g_value_set_string(value, (*C.gchar)(unsafe.Pointer(cStr)))
	case reflect.Int:
		C.g_value_init(value, C.G_TYPE_INT)
		C.g_value_set_int(value, C.gint(v.(int)))
	case reflect.Ptr, reflect.UnsafePointer:
		p := unsafe.Pointer(reflect.ValueOf(v).Pointer())
		if IsObject(p) {
			C.g_value_init(value, C.G_TYPE_OBJECT)
			C.g_value_set_object(value, C.gpointer(p))
		} else {
			panic(fmt.Sprintf("unknown pointer type %v", v))
		}
	default:
		panic(fmt.Sprintf("unknown type %v", v)) //TODO
	}
	return value
}

func IsObject(o unsafe.Pointer) bool {
	return C.is_object(o) == C.TRUE
}
