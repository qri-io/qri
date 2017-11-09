package doggos

import (
	"testing"
)

func TestDoggoNick(t *testing.T) {
	cases := []struct {
		id   string
		nick string
	}{
		{"QmR8iG3S8YJousXRLjitSjdNrmPTyAtTdQtWSX2eMwipzD", "royal_blue_traditional_st_bernard"},
		{"QmaufxVe684Vnm9nUX3ouxPDxTXC6q5t2ZezR5nAnvd588", "sea_blue_australian_cattle_dog"},
		{"QmVJwTCGXo98sVk6b5mfRL7N2tQX7MMKLTfYigdin8KP2t", "pear_australian_terrier"},
	}

	for i, c := range cases {
		got := DoggoNick(c.id)
		if got != c.nick {
			t.Errorf("case %d doggo nick mismatch %s != %s", i, c.nick, got)
			continue
		}
	}
}
