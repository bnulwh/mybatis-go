package orm

import (
	"database/sql"
	"github.com/go-sql-driver/mysql"
	"reflect"
	"testing"
	"time"
)

func Test_newInstance(t *testing.T) {
	type ks struct {
		kind reflect.Kind
		str  string
	}
	mp := map[reflect.Type]ks{
		reflect.TypeOf(""): {
			kind: reflect.Ptr,
			str:  "sql.NullString",
		},
		reflect.TypeOf(true): {
			kind: reflect.Ptr,
			str:  "sql.NullBool",
		},
		reflect.TypeOf(1): {
			kind: reflect.Ptr,
			str:  "sql.NullInt32",
		},
		reflect.TypeOf(int8(1)): {
			kind: reflect.Ptr,
			str:  "sql.NullInt32",
		},
		reflect.TypeOf(int16(1)): {
			kind: reflect.Ptr,
			str:  "sql.NullInt32",
		},
		reflect.TypeOf(int32(1)): {
			kind: reflect.Ptr,
			str:  "sql.NullInt32",
		},
		reflect.TypeOf(uint(1)): {
			kind: reflect.Ptr,
			str:  "sql.NullInt32",
		},
		reflect.TypeOf(uint8(1)): {
			kind: reflect.Ptr,
			str:  "sql.NullInt32",
		},
		reflect.TypeOf(uint16(1)): {
			kind: reflect.Ptr,
			str:  "sql.NullInt32",
		},
		reflect.TypeOf(uint32(1)): {
			kind: reflect.Ptr,
			str:  "sql.NullInt32",
		},
		reflect.TypeOf(int64(1)): {
			kind: reflect.Ptr,
			str:  "sql.NullInt64",
		},
		reflect.TypeOf(uint64(1)): {
			kind: reflect.Ptr,
			str:  "sql.NullInt64",
		},
		reflect.TypeOf(sql.NullInt64{}): {
			kind: reflect.Ptr,
			str:  "sql.NullInt64",
		},
		reflect.TypeOf(float32(0.0)): {
			kind: reflect.Ptr,
			str:  "sql.NullFloat64",
		},
		reflect.TypeOf(float64(0.0)): {
			kind: reflect.Ptr,
			str:  "sql.NullFloat64",
		},
		reflect.TypeOf(time.Now()): {
			kind: reflect.Ptr,
			str:  "sql.NullTime",
		},
		reflect.TypeOf(sql.RawBytes{}): {
			kind: reflect.Ptr,
			str:  "sql.RawBytes",
		},
		reflect.TypeOf(mysql.NullTime{}): {
			kind: reflect.Ptr,
			str:  "mysql.NullTime",
		},
		reflect.TypeOf(ks{}): {
			kind: reflect.Ptr,
			str:  "sql.NullString",
		},
	}
	for k, v := range mp {
		r := newInstance(k)
		ki := reflect.TypeOf(r).Kind()
		si := reflect.ValueOf(r).Elem().Type().String()
		if ki != v.kind || si != v.str {
			t.Errorf("test newInstance failed.type=%v", k)
		}
	}
}

func Test_convertSqlString2String(t *testing.T) {
	r1, err := convertSqlString2String(&sql.NullString{
		String: "test",
		Valid:  true,
	})
	if r1 != "test" || err != nil {
		t.Error("test convertSqlString2String failed.")
	}
	r2, err := convertSqlString2String(&sql.NullString{
		String: "",
		Valid:  false,
	})
	if r2 != "" || err != nil {
		t.Error("test convertSqlString2String failed.")
	}
}

func Test_convertRawBytes2String(t *testing.T) {
	//r1, err := convertRawBytes2String(&sql.RawBytes{
	//	String: "test",
	//	Valid:  true,
	//})
	//if r1 != "test" || err != nil {
	//	t.Error("test convertRawBytes2String failed.")
	//}
	r2, err := convertRawBytes2String(&sql.RawBytes{})
	if r2 != "" || err != nil {
		t.Error("test convertRawBytes2String failed.")
	}

}

func Test_convertSqlBool2Bool(t *testing.T) {
	r1, err := convertSqlBool2Bool(&sql.NullBool{
		Bool:  true,
		Valid: true,
	})
	if !r1 || err != nil {
		t.Error("test convertSqlBool2Bool failed.")
	}
	r2, err := convertSqlBool2Bool(&sql.NullBool{
		Bool:  true,
		Valid: false,
	})
	if r2 || err != nil {
		t.Error("test convertSqlBool2Bool failed.")
	}
}
func Test_convertSqlInt32ToInt(t *testing.T) {
	r1, err := convertSqlInt32ToInt(&sql.NullInt32{
		Int32: 1,
		Valid: true,
	})
	if r1 != 1 || err != nil {
		t.Error("test convertSqlInt32ToInt failed.")
	}
	r2, err := convertSqlInt32ToInt(&sql.NullInt32{
		Int32: 0,
		Valid: false,
	})
	if r2 != 0 || err != nil {
		t.Error("test convertSqlInt32ToInt failed.")
	}
}
func Test_convertSqlInt32ToInt8(t *testing.T) {
	r1, err := convertSqlInt32ToInt8(&sql.NullInt32{
		Int32: 1,
		Valid: true,
	})
	if r1 != 1 || err != nil {
		t.Error("test convertSqlInt32ToInt8 failed.")
	}
	r2, err := convertSqlInt32ToInt8(&sql.NullInt32{
		Int32: 0,
		Valid: false,
	})
	if r2 != 0 || err != nil {
		t.Error("test convertSqlInt32ToInt8 failed.")
	}
}
