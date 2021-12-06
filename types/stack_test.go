package types

import (
	"testing"
)

func TestStack_IsEmpty(t *testing.T) {
	st := newStack()
	r := st.IsEmpty()
	if !r {
		t.Error("test Stack_IsEmpty failed.")
	}
	st.Push("hello")
	r1 := st.IsEmpty()
	if r1 {
		t.Error("test Stack_IsEmpty failed.")
	}
}

func TestStack_Len(t *testing.T) {
	st := newStack()
	r := st.Len()
	if r != 0 {
		t.Error("test Stack_Len failed.")
	}
	st.Push("hello")
	r1 := st.Len()
	if r1 != 1 {
		t.Error("test Stack_Len failed.")
	}
	st.Push("world")
	r2 := st.Len()
	if r2 != 2 {
		t.Error("test Stack_Len failed.")
	}
}

func TestStack_Peak(t *testing.T) {
	st := newStack()
	r := st.Peak()
	if r != nil {
		t.Error("test Stack_Peek failed.")
	}
	st.Push("hello")
	r1 := st.Peak()
	if r1 != "hello" {
		t.Error("test Stack_Peek failed.")
	}
	st.Push("world")
	r2 := st.Peak()
	if r2 != "world" {
		t.Error("test Stack_Peek failed.")
	}
}

func TestStack_Pop(t *testing.T) {
	st := newStack()
	r := st.Pop()
	if r != nil {
		t.Error("test Stack_Pop failed.")
	}
	st.Push("hello")
	st.Push("world")
	r1 := st.Pop()
	if r1 != "world" {
		t.Error("test Stack_Pop failed.")
	}
	r2 := st.Pop()
	if r2 != "hello" {
		t.Error("test Stack_Pop failed.")
	}
	r3 := st.Pop()
	if r3 != nil {
		t.Error("test Stack_Pop failed.")
	}
}

func TestStack_Push(t *testing.T) {
	st := newStack()
	r := st.Len()
	if r != 0 {
		t.Error("test Stack_Len failed.")
	}
	st.Push("hello")
	r1 := st.Len()
	if r1 != 1 {
		t.Error("test Stack_Len failed.")
	}
	st.Push("world")
	r2 := st.Len()
	if r2 != 2 {
		t.Error("test Stack_Len failed.")
	}
}
