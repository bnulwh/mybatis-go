package orm

import (
	"github.com/beevik/etree"
	"os"
	"reflect"
	"testing"
)

func Test_TableStructure_saveToFile(t *testing.T) {
	cs0 := &ColumnStucture{
		Name:    "id",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "id",
		Primary: true,
	}
	cs1 := &ColumnStucture{
		Name:    "name",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "name",
		Primary: false,
	}
	cs2 := &ColumnStucture{
		Name:    "addr_china",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "name",
		Primary: false,
	}
	ts := &TableStructure{
		Columns:       []*ColumnStucture{cs0, cs1, cs2},
		ColumnMap:     map[string]*ColumnStucture{"id": cs0, "name": cs1, "addr_china": cs2},
		Table:         "test",
		PrimaryColumn: cs0,
	}
	path := "test.xml"
	err := ts.saveToFile(path)
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
	cs0 := &ColumnStucture{
		Name:    "id",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "id",
		Primary: true,
	}
	cs1 := &ColumnStucture{
		Name:    "name",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "name",
		Primary: false,
	}
	cs2 := &ColumnStucture{
		Name:    "addr_china",
		Type:    reflect.TypeOf(""),
		DbType:  "varchar",
		Comment: "name",
		Primary: false,
	}
	ts := &TableStructure{
		Columns:       []*ColumnStucture{cs0, cs1, cs2},
		ColumnMap:     map[string]*ColumnStucture{"id": cs0, "name": cs1, "addr_china": cs2},
		Table:         "test",
		PrimaryColumn: cs0,
	}
	doc := etree.NewDocument()
	mp := ts.createMapper(doc)
	if mp == nil {
		t.Errorf("Test TableStructure createMapper failed.")
	}
}
