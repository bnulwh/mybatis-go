package types

import "testing"

func Test_toGolangType(t *testing.T) {
	mp := map[string]string{
		"STRING": "string", "VARCHAR": "string",
		"BOOLEAN": "bool", "BOOL": "bool",
		"INT": "int32", "INTEGER": "int32", "INT8": "int32", "INT16": "int32", "INT32": "int32",
		"INT64": "int64",
		"UINT":  "uint32", "UINT8": "uint32", "UINT16": "uint32", "UINT32": "uint32",
		"UINT64": "uint64",
		"FLOAT":  "float32", "FLOAT32": "float32",
		"FLOAT64": "float64", "DOUBLE": "float64",
		"TIME": "time.Time", "TIMESTAMP": "time.Time",
		"LIST": "[]interface{}", "ARRAY": "[]interface{}", "ARRAYLIST": "[]interface{}", "SLICE": "[]interface{}",
		"MAP": "map[string]interface{}", "HASHMAP": "map[string]interface{}", "TREEMAP": "map[string]interface{}",
	}
	for k, v := range mp {
		r := toGolangType(k)
		if r != v {
			t.Errorf("test toGolangType failed.k= %v v=%v r=%v", k, v, r)
		}
	}
}
