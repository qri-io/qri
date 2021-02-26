package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/qri-io/qri/auth/token"
	"github.com/qri-io/qri/profile"
)

type AccessMethods struct {
	inst *Instance
}

func NewAccessMethods(inst *Instance) *AccessMethods {
	return &AccessMethods{inst: inst}
}

type CreateTokenParams struct {
	GrantorUsername  string
	GrantorProfileID string
	TTL              time.Duration
}

func (m *AccessMethods) CreateToken(ctx context.Context, p *CreateTokenParams) (string, error) {
	var (
		grantor *profile.Profile
		err     error
	)

	if p.GrantorUsername == "" {
		id, err := profile.IDB58Decode(p.GrantorProfileID)
		if err != nil {
			return "", err
		}
		if grantor, err = m.inst.profiles.GetProfile(id); err != nil {
			return "", err
		}
	} else if p.GrantorUsername != "" {
		if grantor, err = profile.ResolveUsername(m.inst.profiles, p.GrantorUsername); err != nil {
			return "", err
		}
	} else {
		return "", fmt.Errorf("either grantor username or profile is required")
	}

	pk := grantor.PrivKey
	if pk == nil {
		return "", fmt.Errorf("cannot create token for profile %s (id: %s), private key is required", grantor.Peername, grantor.ID.String())
	}

	src, err := token.NewPrivKeySource(pk)
	if err != nil {
		return "", err
	}

	if p.TTL == 0 {
		p.TTL = token.DefaultTokenTTL
	}

	return src.CreateToken(grantor.ID.String(), grantor.Peername, p.TTL)
}
