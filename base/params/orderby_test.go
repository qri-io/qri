package params

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOrderString(t *testing.T) {
	expect := "+name"
	o := &Order{Key: "name", Direction: OrderASC}
	if expect != o.String() {
		t.Errorf("String mismatch, expected %s, got %s", expect, o.String())
	}
}

func TestOrderByString(t *testing.T) {
	expect := "+name,-updated"
	o := OrderBy{
		{Key: "name", Direction: OrderASC},
		{Key: "updated", Direction: OrderDESC},
	}
	if expect != o.String() {
		t.Errorf("String mismatch, expected %s, got %s", expect, o.String())
	}
}

func TestNewOrder(t *testing.T) {
	got := NewOrder("", OrderASC)
	if got != nil {
		t.Errorf("empty key should return nil order")
		return
	}
	expect := &Order{Key: "name", Direction: OrderASC}
	got = NewOrder("name", OrderASC)
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("Order mismatch (+want,-got):\n%s", diff)
		return
	}
}

func TestNewOrderFromString(t *testing.T) {
	got := NewOrderFromString("")
	if got != nil {
		t.Errorf("empty string should return nil order")
		return
	}

	cases := []struct {
		orderStr string
		expect   *Order
	}{
		{"+name", &Order{Key: "name", Direction: OrderASC}},
		{"-updated", &Order{Key: "updated", Direction: OrderDESC}},
		{"username", &Order{Key: "username", Direction: OrderASC}},
	}
	for _, c := range cases {
		got := NewOrderFromString(c.orderStr)
		if diff := cmp.Diff(c.expect, got); diff != "" {
			t.Errorf("case %q mismatch (+want,-got):\n%s", c.orderStr, diff)
			return
		}
	}
}

func TestOrderByFromString(t *testing.T) {
	got := NewOrderByFromString("")
	if len(got) != 0 {
		t.Errorf("empty string should return empty OrderBy")
	}

	expect := OrderBy{
		{Key: "updated", Direction: OrderASC},
		{Key: "name", Direction: OrderDESC},
		{Key: "username", Direction: OrderASC},
	}
	got = NewOrderByFromString("+updated,-name,username")
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("OrderBy mismatch (+want,-got):\n%s", diff)
		return
	}
}
