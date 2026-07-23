package orm

import (
	"github.com/beevik/etree"
	"github.com/bnulwh/mybatis-go/types"
	"os"
	"reflect"
	"testing"
)

func Test_TableStructure_saveToFile(t *testing.T) {
	cs0 := &types.ColumnStructure{
		Name:    "id",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "id",
		Primary: true,
	}
	cs1 := &types.ColumnStructure{
		Name:    "name",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "name",
		Primary: false,
	}
	cs2 := &types.ColumnStructure{
		Name:    "addr_china",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "name",
		Primary: false,
	}
	ts := &types.TableStructure{
		Columns:       []*types.ColumnStructure{cs0, cs1, cs2},
		ColumnMap:     map[string]*types.ColumnStructure{"id": cs0, "name": cs1, "addr_china": cs2},
		Table:         "test",
		PrimaryColumn: cs0,
	}
	path := "test.xml"
	err := ts.SaveToFile(path, "")
	if err != nil {
		t.Errorf("Test TableStructure saveToFile failed. %v", err)
	}
	_, err = os.Stat(path)
	if err != nil {
		if !os.IsExist(err) {
			//t.Skip()
			t.Errorf("Test TableStructure saveToFile failed. %v", err)
		}
	}
}

func Test_TableStructure_createMapper(t *testing.T) {
	cs0 := &types.ColumnStructure{
		Name:    "id",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "id",
		Primary: true,
	}
	cs1 := &types.ColumnStructure{
		Name:    "name",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "name",
		Primary: false,
	}
	cs2 := &types.ColumnStructure{
		Name:    "addr_china",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "name",
		Primary: false,
	}
	ts := &types.TableStructure{
		Columns:       []*types.ColumnStructure{cs0, cs1, cs2},
		ColumnMap:     map[string]*types.ColumnStructure{"id": cs0, "name": cs1, "addr_china": cs2},
		Table:         "test",
		PrimaryColumn: cs0,
	}
	doc := etree.NewDocument()
	mp := ts.CreateMapper(doc, "")
	if mp == nil {
		t.Errorf("Test TableStructure createMapper failed.")
	}
}
