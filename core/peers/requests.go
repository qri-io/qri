package peers

import (
	"fmt"
	"github.com/qri-io/qri/repo"
	"github.com/qri-io/qri/repo/profile"
)

func NewRequests(r repo.Repo) *Requests {
	return &Requests{
		repo: r,
	}
}

type Requests struct {
	repo repo.Repo
}

type ListParams struct {
	OrderBy string
	Limit   int
	Offset  int
}

func (d *Requests) List(p *ListParams, res *[]*profile.Profile) error {
	replies := make([]*profile.Profile, p.Limit)
	i := 0
	peers, err := d.repo.Peers()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	for _, repo := range peers {
		if i >= p.Limit {
			break
		}
		replies[i] = repo.Profile
		i++
	}
	*res = replies[:i]
	return nil
}

type GetParams struct {
	Username string
	Name     string
	Hash     string
}

func (d *Requests) Get(p *GetParams, res *profile.Profile) error {
	peers, err := d.repo.Peers()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	for name, repo := range peers {
		if p.Hash == name ||
			p.Username == repo.Profile.Username {
			*res = *repo.Profile
		}
	}

	if res != nil {
		return nil
	}

	// TODO - ErrNotFound plz
	return fmt.Errorf("Not Found")
}
