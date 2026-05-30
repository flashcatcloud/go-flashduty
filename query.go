package flashduty

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// addQueryParams appends opt's `url`-tagged struct fields to path as query
// parameters. A field tagged `url:"name,omitempty"` is skipped when it holds
// its zero value; `url:"-"` and untagged fields are ignored. This is a small,
// dependency-free analogue of go-querystring used by the generated GET methods.
func addQueryParams(path string, opt any) (string, error) {
	if opt == nil {
		return path, nil
	}
	v := reflect.ValueOf(opt)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return path, nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return path, fmt.Errorf("query options must be a struct, got %s", v.Kind())
	}

	q := url.Values{}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("url")
		if tag == "" || tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if name == "" {
			continue
		}
		fv := v.Field(i)
		if strings.Contains(opts, "omitempty") && fv.IsZero() {
			continue
		}
		switch fv.Kind() {
		case reflect.String:
			q.Set(name, fv.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			q.Set(name, strconv.FormatInt(fv.Int(), 10))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			q.Set(name, strconv.FormatUint(fv.Uint(), 10))
		case reflect.Bool:
			q.Set(name, strconv.FormatBool(fv.Bool()))
		case reflect.Float32, reflect.Float64:
			q.Set(name, strconv.FormatFloat(fv.Float(), 'g', -1, 64))
		case reflect.Slice, reflect.Array:
			for j := 0; j < fv.Len(); j++ {
				q.Add(name, fmt.Sprint(fv.Index(j).Interface()))
			}
		default:
			q.Set(name, fmt.Sprint(fv.Interface()))
		}
	}

	if len(q) == 0 {
		return path, nil
	}
	if strings.Contains(path, "?") {
		return path + "&" + q.Encode(), nil
	}
	return path + "?" + q.Encode(), nil
}
