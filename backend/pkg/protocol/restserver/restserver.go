package restserver

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

var (
	typeOfBytes = reflect.TypeOf([]byte(nil))
	rawJSONMIME = "application/raw-json" // made-up MIME type for webhook
)

type rawJSONPb struct {
	*runtime.JSONPb
}

func (*rawJSONPb) ContentType(v interface{}) string {
	return rawJSONMIME
}

func (*rawJSONPb) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(v interface{}) error {
		raw, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		rv := reflect.ValueOf(v)

		if rv.Kind() != reflect.Ptr {
			return fmt.Errorf("%T is not a pointer", v)
		}

		rv = rv.Elem()
		if rv.Type() != typeOfBytes {
			return fmt.Errorf("type must be []byte but got %T", v)
		}

		rv.Set(reflect.ValueOf(raw))
		return nil
	})
}

// CustomMIME middleware for intercepting 'webhook' in URLs, and converting
// its content type to custom one
func CustomMIME(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "webhook") {
			r.Header.Set("Content-Type", rawJSONMIME)
		}
		h.ServeHTTP(w, r)
	})
}

func NewRESTServer() *runtime.ServeMux {
	jsonpb := &runtime.JSONPb{}

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(rawJSONMIME, &rawJSONPb{jsonpb}), // if content-type == "application/raw-json"
		runtime.WithMarshalerOption(runtime.MIMEWildcard, jsonpb),    // all other content-types
	)
	return mux
}
