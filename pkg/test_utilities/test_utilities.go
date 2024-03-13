package test_utilities

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"unsafe"

	"net/http/httptest"
)

////////////////////////////////////////////////////////////////////////////
// Test utilities
////////////////////////////////////////////////////////////////////////////

// setFieldValue is only for testing
func setFieldValue(target any, fieldName string, value any) {
	rv := reflect.ValueOf(target)
	for rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem()
	}
	if !rv.CanAddr() {
		panic("target must be addressable")
	}
	if rv.Kind() != reflect.Struct {
		panic(fmt.Sprintf(
			"unable to set the '%s' field value of the type %T, target must be a struct",
			fieldName,
			target,
		))
	}
	rf := rv.FieldByName(fieldName)

	reflect.NewAt(rf.Type(), unsafe.Pointer(rf.UnsafeAddr())).Elem().Set(reflect.ValueOf(value))
}

type TestResponseRecorder struct {
	Code      int
	HeaderMap http.Header
	Body      *bytes.Buffer
	Flushed   bool

	Result      *http.Response // cache of Result's return value
	SnapHeader  http.Header    // snapshot of HeaderMap at first Write
	WroteHeader bool
}

func (trr TestResponseRecorder) ToResponseRecorder() *httptest.ResponseRecorder {
	rr := &httptest.ResponseRecorder{
		Code:      trr.Code,
		HeaderMap: trr.HeaderMap,
		Body:      trr.Body,
		Flushed:   trr.Flushed,
	}
	if trr.Result != nil {
		setFieldValue(rr, "result", trr.Result)
	}
	if trr.SnapHeader != nil {
		setFieldValue(rr, "snapHeader", trr.SnapHeader)
	}
	setFieldValue(rr, "wroteHeader", trr.WroteHeader)
	return rr
}
