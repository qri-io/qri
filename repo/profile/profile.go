package profile

import (
	"time"
)

type Profile struct {
	Id string `json:"id"`
	// Created timestamp
	Created time.Time `json:"created,omitempty"`
	// Updated timestamp
	Updated time.Time `json:"updated,omitempty"`
	// handle for the user. min 1 character, max 80. composed of [_,-,a-z,A-Z,1-9]
	Username string `json:"username"`
	// specifies weather this is a user or an organization
	Type UserType `json:"type"`
	// user's email address
	Email string `json:"email"`
	// user name field. could be first[space]last, but not strictly enforced
	Name string `json:"name"`
	// user-filled description of self
	Description string `json:"description"`
	// url this user wants the world to click
	HomeUrl string `json:"homeUrl"`
	// color this user likes to use as their theme color
	Color string `json:"color"`
	// url for their thumbnail
	ThumbUrl string `json:"thumbUrl"`
	// profile photo url
	ProfileUrl string `json:"profileUrl"`
	// users's twitter handle
	Twitter string `json:"twitter"`
	// often users get auto-generated based on IP for rate lmiting & stuff
	// this flag tracks that.
	// TODO - for this to be useful it'll need to be Exported
	Anonymous bool `json:",omitempty"`
}
