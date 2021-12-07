package utils

import (
	"reflect"
	"testing"
)

func Test_change2String(t *testing.T) {
	r, err := change2String("abc")
	if r != "abc" || err != nil {
		t.Error("test change2String failed.")
	}
	r1, err := change2String(1)
	if r1 != "1" || err != nil {
		t.Error("test change2String failed.")
	}
}

func Test_change2Bool(t *testing.T) {
	r, err := change2Bool("abc")
	if r || err == nil {
		t.Errorf("test change2Bool failed. %v", err)
	}
	r1, err := change2Bool(1)
	//fmt.Println(r1)
	if !r1 || err != nil {
		t.Errorf("test change2Bool failed. %v", err)
	}
	r2, err := change2Bool(0)
	//fmt.Println(r2)
	if r2 || err != nil {
		t.Errorf("test change2Bool failed. %v", err)
	}
	r3, err := change2Bool("true")
	//fmt.Println(r3)
	if !r3 || err != nil {
		t.Errorf("test change2Bool failed. %v", err)
	}
	r4, err := change2Bool("f")
	//fmt.Println(r4)
	if r4 || err != nil {
		t.Errorf("test change2Bool failed. %v", err)
	}
	r5, err := change2Bool(true)
	//fmt.Println(r3)
	if !r5 || err != nil {
		t.Errorf("test change2Bool failed. %v", err)
	}
	r6, err := change2Bool(false)
	//fmt.Println(r4)
	if r6 || err != nil {
		t.Errorf("test change2Bool failed. %v", err)
	}
}

func Test_change2Int(t *testing.T) {
	r1, err := change2Int(1)
	if r1 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r2, err := change2Int(int8(1))
	if r2 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r3, err := change2Int(int16(1))
	if r3 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r4, err := change2Int(int32(1))
	if r4 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r5, err := change2Int(int64(1))
	if r5 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r6, err := change2Int("1")
	if r6 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r7, err := change2Int("0x1")
	if r7 != 0 || err == nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r8, err := change2Int("a")
	if r8 != 0 || err == nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r9, err := change2Int(uint(1))
	if r9 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r10, err := change2Int(uint8(1))
	if r10 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r11, err := change2Int(uint16(1))
	if r11 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r12, err := change2Int(uint32(1))
	if r12 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
	r13, err := change2Int(uint64(1))
	if r13 != 1 || err != nil {
		t.Errorf("test change2Int failed. %v", err)
	}
}

func Test_change2Int8(t *testing.T) {
	r1, err := change2Int8(1)
	if r1 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r2, err := change2Int8(int8(1))
	if r2 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r3, err := change2Int8(int16(1))
	if r3 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r4, err := change2Int8(int32(1))
	if r4 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r5, err := change2Int8(int64(1))
	if r5 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r6, err := change2Int8("1")
	if r6 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r7, err := change2Int8("0x1")
	if r7 != int8(0) || err == nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r8, err := change2Int8("a")
	if r8 != int8(0) || err == nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r9, err := change2Int8(uint(1))
	if r9 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r10, err := change2Int8(uint8(1))
	if r10 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r11, err := change2Int8(uint16(1))
	if r11 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r12, err := change2Int8(uint32(1))
	if r12 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
	r13, err := change2Int8(uint64(1))
	if r13 != int8(1) || err != nil {
		t.Errorf("test change2Int8 failed. %v", err)
	}
}
func Test_change2Int16(t *testing.T) {
	r1, err := change2Int16(1)
	if r1 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r2, err := change2Int16(int8(1))
	if r2 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r3, err := change2Int16(int16(1))
	if r3 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r4, err := change2Int16(int32(1))
	if r4 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r5, err := change2Int16(int64(1))
	if r5 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r6, err := change2Int16("1")
	if r6 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r7, err := change2Int16("0x1")
	if r7 != int16(0) || err == nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r8, err := change2Int16("a")
	if r8 != int16(0) || err == nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r9, err := change2Int16(uint(1))
	if r9 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r10, err := change2Int16(uint8(1))
	if r10 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r11, err := change2Int16(uint16(1))
	if r11 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r12, err := change2Int16(uint32(1))
	if r12 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
	r13, err := change2Int16(uint64(1))
	if r13 != int16(1) || err != nil {
		t.Errorf("test change2Int16 failed. %v", err)
	}
}
func Test_change2Int32(t *testing.T) {
	r1, err := change2Int32(1)
	if r1 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r2, err := change2Int32(int8(1))
	if r2 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r3, err := change2Int32(int16(1))
	if r3 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r4, err := change2Int32(int32(1))
	if r4 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r5, err := change2Int32(int64(1))
	if r5 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r6, err := change2Int32("1")
	if r6 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r7, err := change2Int32("0x1")
	if r7 != int32(0) || err == nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r8, err := change2Int32("a")
	if r8 != int32(0) || err == nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r9, err := change2Int32(uint(1))
	if r9 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r10, err := change2Int32(uint8(1))
	if r10 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r11, err := change2Int32(uint16(1))
	if r11 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r12, err := change2Int32(uint32(1))
	if r12 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
	r13, err := change2Int32(uint64(1))
	if r13 != int32(1) || err != nil {
		t.Errorf("test change2Int32 failed. %v", err)
	}
}

func Test_change2Int64(t *testing.T) {
	r1, err := change2Int64(1)
	if r1 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r2, err := change2Int64(int8(1))
	if r2 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r3, err := change2Int64(int16(1))
	if r3 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r4, err := change2Int64(int32(1))
	if r4 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r5, err := change2Int64(int64(1))
	if r5 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r6, err := change2Int64("1")
	if r6 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r7, err := change2Int64("0x1")
	if r7 != int64(0) || err == nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r8, err := change2Int64("a")
	if r8 != int64(0) || err == nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r9, err := change2Int64(uint(1))
	if r9 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r10, err := change2Int64(uint8(1))
	if r10 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r11, err := change2Int64(uint16(1))
	if r11 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r12, err := change2Int64(uint32(1))
	if r12 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r13, err := change2Int64(uint64(1))
	if r13 != int64(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
}

func Test_change2UInt(t *testing.T) {
	r1, err := change2UInt(1)
	if r1 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r2, err := change2UInt(int8(1))
	if r2 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r3, err := change2UInt(int16(1))
	if r3 != uint(1) || err != nil {
		t.Errorf("test change2Int64 failed. %v", err)
	}
	r4, err := change2UInt(int32(1))
	if r4 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r5, err := change2UInt(int64(1))
	if r5 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r6, err := change2UInt("1")
	if r6 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r7, err := change2UInt("0x1")
	if r7 != uint(0) || err == nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r8, err := change2UInt("a")
	if r8 != uint(0) || err == nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r9, err := change2UInt(uint(1))
	if r9 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r10, err := change2UInt(uint8(1))
	if r10 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r11, err := change2UInt(uint16(1))
	if r11 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r12, err := change2UInt(uint32(1))
	if r12 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
	r13, err := change2UInt(uint64(1))
	if r13 != uint(1) || err != nil {
		t.Errorf("test change2UInt failed. %v", err)
	}
}

func Test_change2UInt8(t *testing.T) {
	r1, err := change2UInt8(1)
	if r1 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r2, err := change2UInt8(int8(1))
	if r2 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r3, err := change2UInt8(int16(1))
	if r3 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r4, err := change2UInt8(int32(1))
	if r4 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r5, err := change2UInt8(int64(1))
	if r5 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r6, err := change2UInt8("1")
	if r6 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r7, err := change2UInt8("0x1")
	if r7 != uint8(0) || err == nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r8, err := change2UInt8("a")
	if r8 != uint8(0) || err == nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r9, err := change2UInt8(uint(1))
	if r9 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r10, err := change2UInt8(uint8(1))
	if r10 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r11, err := change2UInt8(uint16(1))
	if r11 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r12, err := change2UInt8(uint32(1))
	if r12 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
	r13, err := change2UInt8(uint64(1))
	if r13 != uint8(1) || err != nil {
		t.Errorf("test change2UInt8 failed. %v", err)
	}
}

func Test_change2UInt16(t *testing.T) {
	r1, err := change2UInt16(1)
	if r1 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r2, err := change2UInt16(int8(1))
	if r2 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r3, err := change2UInt16(int16(1))
	if r3 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r4, err := change2UInt16(int32(1))
	if r4 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r5, err := change2UInt16(int64(1))
	if r5 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r6, err := change2UInt16("1")
	if r6 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r7, err := change2UInt16("0x1")
	if r7 != uint16(0) || err == nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r8, err := change2UInt16("a")
	if r8 != uint16(0) || err == nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r9, err := change2UInt16(uint(1))
	if r9 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r10, err := change2UInt16(uint8(1))
	if r10 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r11, err := change2UInt16(uint16(1))
	if r11 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r12, err := change2UInt16(uint32(1))
	if r12 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
	r13, err := change2UInt16(uint64(1))
	if r13 != uint16(1) || err != nil {
		t.Errorf("test change2UInt16 failed. %v", err)
	}
}

func Test_change2UInt32(t *testing.T) {
	r1, err := change2UInt32(1)
	if r1 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r2, err := change2UInt32(int8(1))
	if r2 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r3, err := change2UInt32(int16(1))
	if r3 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r4, err := change2UInt32(int32(1))
	if r4 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r5, err := change2UInt32(int64(1))
	if r5 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r6, err := change2UInt32("1")
	if r6 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r7, err := change2UInt32("0x1")
	if r7 != uint32(0) || err == nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r8, err := change2UInt32("a")
	if r8 != uint32(0) || err == nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r9, err := change2UInt32(uint(1))
	if r9 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r10, err := change2UInt32(uint8(1))
	if r10 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r11, err := change2UInt32(uint16(1))
	if r11 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r12, err := change2UInt32(uint32(1))
	if r12 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
	r13, err := change2UInt32(uint64(1))
	if r13 != uint32(1) || err != nil {
		t.Errorf("test change2UInt32 failed. %v", err)
	}
}

func Test_change2UInt64(t *testing.T) {
	r1, err := change2UInt64(1)
	if r1 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r2, err := change2UInt64(int8(1))
	if r2 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r3, err := change2UInt64(int16(1))
	if r3 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r4, err := change2UInt64(int32(1))
	if r4 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r5, err := change2UInt64(int64(1))
	if r5 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r6, err := change2UInt64("1")
	if r6 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r7, err := change2UInt64("0x1")
	if r7 != uint64(0) || err == nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r8, err := change2UInt64("a")
	if r8 != uint64(0) || err == nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r9, err := change2UInt64(uint(1))
	if r9 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r10, err := change2UInt64(uint8(1))
	if r10 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r11, err := change2UInt64(uint16(1))
	if r11 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r12, err := change2UInt64(uint32(1))
	if r12 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
	r13, err := change2UInt64(uint64(1))
	if r13 != uint64(1) || err != nil {
		t.Errorf("test change2UInt64 failed. %v", err)
	}
}

func Test_change2Float32(t *testing.T) {
	r1, err := change2Float32(1)
	//fmt.Println(r1)
	if r1 != 1 || err != nil || reflect.TypeOf(r1).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r2, err := change2Float32(int8(1))
	if r2 != 1 || err != nil || reflect.TypeOf(r2).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r3, err := change2Float32(int16(1))
	if r3 != 1 || err != nil || reflect.TypeOf(r3).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r4, err := change2Float32(int32(1))
	if r4 != 1 || err != nil || reflect.TypeOf(r4).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r5, err := change2Float32(int64(1))
	if r5 != 1 || err != nil || reflect.TypeOf(r5).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r6, err := change2Float32("1")
	if r6 != 1 || err != nil || reflect.TypeOf(r6).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r7, err := change2Float32("0x1")
	if r7 != 0 || err == nil || reflect.TypeOf(r7).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r8, err := change2Float32("a")
	if r8 != 0 || err == nil || reflect.TypeOf(r8).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r9, err := change2Float32(uint(1))
	if r9 != 1 || err != nil || reflect.TypeOf(r9).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r10, err := change2Float32(uint8(1))
	if r10 != 1 || err != nil || reflect.TypeOf(r10).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r11, err := change2Float32(uint16(1))
	if r11 != 1 || err != nil || reflect.TypeOf(r11).Kind() != reflect.Float32 {
		t.Errorf("test change2Int failed. %v", err)
	}
	r12, err := change2Float32(uint32(1))
	if r12 != 1 || err != nil || reflect.TypeOf(r12).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r13, err := change2Float32(uint64(1))
	if r13 != 1 || err != nil || reflect.TypeOf(r13).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r14, err := change2Float32(1.0)
	if r14 != 1 || err != nil || reflect.TypeOf(r14).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
	r15, err := change2Float32(float32(1.0))
	if r15 != 1 || err != nil || reflect.TypeOf(r15).Kind() != reflect.Float32 {
		t.Errorf("test change2Float32 failed. %v", err)
	}
}

func Test_change2Float64(t *testing.T) {
	r1, err := change2Float64(1)
	//fmt.Println(r1)
	if r1 != 1 || err != nil || reflect.TypeOf(r1).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r2, err := change2Float64(int8(1))
	if r2 != 1 || err != nil || reflect.TypeOf(r2).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r3, err := change2Float64(int16(1))
	if r3 != 1 || err != nil || reflect.TypeOf(r3).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r4, err := change2Float64(int32(1))
	if r4 != 1 || err != nil || reflect.TypeOf(r4).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r5, err := change2Float64(int64(1))
	if r5 != 1 || err != nil || reflect.TypeOf(r5).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r6, err := change2Float64("1")
	if r6 != 1 || err != nil || reflect.TypeOf(r6).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r7, err := change2Float64("0x1")
	if r7 != 0 || err == nil || reflect.TypeOf(r7).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r8, err := change2Float64("a")
	if r8 != 0 || err == nil || reflect.TypeOf(r8).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r9, err := change2Float64(uint(1))
	if r9 != 1 || err != nil || reflect.TypeOf(r9).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r10, err := change2Float64(uint8(1))
	if r10 != 1 || err != nil || reflect.TypeOf(r10).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r11, err := change2Float64(uint16(1))
	if r11 != 1 || err != nil || reflect.TypeOf(r11).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r12, err := change2Float64(uint32(1))
	if r12 != 1 || err != nil || reflect.TypeOf(r12).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r13, err := change2Float64(uint64(1))
	if r13 != 1 || err != nil || reflect.TypeOf(r13).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r14, err := change2Float64(1.0)
	if r14 != 1 || err != nil || reflect.TypeOf(r14).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
	r15, err := change2Float64(float32(1.0))
	if r15 != 1 || err != nil || reflect.TypeOf(r15).Kind() != reflect.Float64 {
		t.Errorf("test change2Float64 failed. %v", err)
	}
}
