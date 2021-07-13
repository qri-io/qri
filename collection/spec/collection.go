package spec

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
	profiletest "github.com/qri-io/qri/profile/test"
)

// Constructor is a function for creating collections, used by spec tests
type Constructor func(ctx context.Context, bus event.Bus) (collection.Set, error)

// AssertWritableCollectionSpec defines expected behaviours for a Writable
// collection implementation
func AssertWritableCollectionSpec(t *testing.T, constructor Constructor) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus := event.NewBus(ctx)

	c, err := constructor(ctx, bus)
	if err != nil {
		t.Fatal(err)
	}

	ec, ok := c.(collection.WritableSet)
	if !ok {
		t.Fatal("construtor did not return a writable collection set")
	}

	kermit := profiletest.GetProfile("kermit")
	missPiggy := profiletest.GetProfile("miss_piggy")

	t.Run("empty_list", func(t *testing.T) {
		res, err := ec.List(ctx, missPiggy.ID, params.ListAll)
		if err == nil {
			t.Fatalf("expected error listing unknown profile, got nil")
		}

		if len(res) != 0 {
			t.Errorf("expected listing to return 0 items. got: %d", len(res))
		}
	})

	t.Run("put", func(t *testing.T) {
		if err := ec.Put(ctx, kermit.ID); err != nil {
			t.Error("expected put with empty item list NOT to error")
		}

		badItems := []struct {
			problem string
			item    dsref.VersionInfo
		}{
			{"empty", dsref.VersionInfo{}},
			{"no InitID", dsref.VersionInfo{ProfileID: kermit.ID.String()}},
			{"no profileID", dsref.VersionInfo{InitID: "init_id"}},
			{"no name", dsref.VersionInfo{InitID: "init_id", ProfileID: kermit.ID.String()}},
		}

		for _, bad := range badItems {
			t.Run(fmt.Sprintf("bad_item_%s", bad.problem), func(t *testing.T) {
				if err := ec.Put(ctx, kermit.ID, bad.item); err == nil {
					t.Error("expected error, got nil")
				}
			})
		}

		err := ec.Put(ctx, kermit.ID,
			dsref.VersionInfo{
				ProfileID:  kermit.ID.String(),
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			dsref.VersionInfo{
				ProfileID:  kermit.ID.String(),
				InitID:     "muppet_names_and_ages_init_id",
				Username:   "kermit",
				Name:       "muppet_names_and_ages",
				CommitTime: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		)

		if err != nil {
			t.Fatalf("error adding items: %s", err)
		}

		err = ec.Put(ctx, missPiggy.ID,
			dsref.VersionInfo{
				ProfileID:  missPiggy.ID.String(),
				InitID:     "secret_muppet_friends_init_id",
				Username:   "miss_piggy",
				Name:       "secret_muppet_friends",
				CommitTime: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
			dsref.VersionInfo{
				ProfileID:  missPiggy.ID.String(),
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			dsref.VersionInfo{
				ProfileID:  missPiggy.ID.String(),
				InitID:     "famous_muppets_init_id",
				Username:   "famous_muppets",
				Name:       "famous_muppets",
				CommitTime: time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
			})

		if err != nil {
			t.Fatalf("error adding items: %s", err)
		}
	})

	t.Run("list", func(t *testing.T) {
		assertCollectionList(ctx, t, kermit, params.ListAll, ec, []dsref.VersionInfo{
			{
				ProfileID:  kermit.ID.String(),
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ProfileID:  kermit.ID.String(),
				InitID:     "muppet_names_and_ages_init_id",
				Username:   "kermit",
				Name:       "muppet_names_and_ages",
				CommitTime: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		})

		assertCollectionList(ctx, t, missPiggy, params.ListAll, ec, []dsref.VersionInfo{
			{
				ProfileID:  missPiggy.ID.String(),
				InitID:     "famous_muppets_init_id",
				Username:   "famous_muppets",
				Name:       "famous_muppets",
				CommitTime: time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
			},
			{
				ProfileID:  missPiggy.ID.String(),
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ProfileID:  missPiggy.ID.String(),
				InitID:     "secret_muppet_friends_init_id",
				Username:   "miss_piggy",
				Name:       "secret_muppet_friends",
				CommitTime: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		})
	})

	t.Run("delete", func(t *testing.T) {
		if err := ec.Delete(ctx, missPiggy.ID); err != nil {
			t.Errorf("expected delete with no initIDs to not fail")
		}

		badCases := []struct {
			reason    string
			profileID profile.ID
			ids       []string
		}{
			{"missing ID", kermit.ID, []string{"unknown"}},
		}

		for _, bad := range badCases {
			t.Run(fmt.Sprintf("bad_case_%s", bad.reason), func(t *testing.T) {
				if err := ec.Delete(ctx, bad.profileID, bad.ids...); err == nil {
					t.Errorf("expected bad case to error. got nil")
				}
			})
		}

		err := ec.Delete(ctx, missPiggy.ID,
			"famous_muppets_init_id",
			"muppet_names_init_id",
			"secret_muppet_friends_init_id",
		)
		if err != nil {
			t.Errorf("unexpected error deleting items: %s", err)
		}

		assertCollectionList(ctx, t, missPiggy, params.ListAll, ec, []dsref.VersionInfo{})
	})
}

// AssertCollectionEventListenerSpec defines expected behaviours for collections
// that use event subscriptions to update state.
// Event Listener specs are optional.
func AssertCollectionEventListenerSpec(t *testing.T, constructor Constructor) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus := event.NewBus(ctx)
	c, err := constructor(ctx, bus)
	if err != nil {
		t.Fatal(err)
	}

	kermit := profiletest.GetProfile("kermit")
	missPiggy := profiletest.GetProfile("miss_piggy")

	t.Run("empty_unlimited_list", func(t *testing.T) {
		_, err := c.List(ctx, profile.ID(""), params.ListAll)
		if err == nil {
			t.Fatalf("listing without providing any keyIDs should error: %q", err)
		}
	})

	t.Run("user_1_manual_datasets", func(t *testing.T) {
		muppetNamesInitID := "initID"
		muppetNamesName1 := "muppet_names"
		muppetNamesName2 := "muppet_names_and_ages"

		// simulate name initialization, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetNameInit, event.DsChange{
			InitID:     muppetNamesInitID,
			Username:   kermit.Peername,
			ProfileID:  kermit.ID.String(),
			PrettyName: muppetNamesName1,
			Info: &dsref.VersionInfo{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID.String(),
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
			},
		})

		expect := []dsref.VersionInfo{
			{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID.String(),
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
			},
		}
		assertCollectionList(ctx, t, kermit, params.ListAll, c, expect)

		// simulate version creation, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetCommitChange, event.DsChange{
			InitID:    muppetNamesInitID,
			ProfileID: kermit.ID.String(),
			TopIndex:  2,
			HeadRef:   "/mem/PathToMuppetNamesVersionOne",
			Info: &dsref.VersionInfo{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID.String(),
				Path:      "/mem/PathToMuppetNamesVersionOne",
				Name:      muppetNamesName1,
			},
		})

		expect = []dsref.VersionInfo{
			{
				InitID:      muppetNamesInitID,
				ProfileID:   kermit.ID.String(),
				Username:    kermit.Peername,
				Name:        muppetNamesName1,
				NumVersions: 2,
				Path:        "/mem/PathToMuppetNamesVersionOne",
			},
		}
		assertCollectionList(ctx, t, kermit, params.ListAll, c, expect)

		// simulate dataset renaming, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetRename, event.DsChange{
			InitID:     muppetNamesInitID,
			PrettyName: muppetNamesName2,
		})

		expect = []dsref.VersionInfo{
			{
				InitID:      muppetNamesInitID,
				ProfileID:   kermit.ID.String(),
				Username:    kermit.Peername,
				Name:        muppetNamesName2,
				NumVersions: 2,
				Path:        "/mem/PathToMuppetNamesVersionOne",
			},
		}
		assertCollectionList(ctx, t, kermit, params.ListAll, c, expect)

		mustPublish(ctx, t, bus, event.ETDatasetDeleteAll, event.DsChange{
			InitID:     muppetNamesInitID,
			PrettyName: muppetNamesName2,
		})

		expect = []dsref.VersionInfo{}
		assertCollectionList(ctx, t, kermit, params.ListAll, c, expect)

		// TODO (b5): create a second dataset, use different timestamps for both,
		// assert default ordering of datasets
	})

	t.Run("user_1_ordering_and_filtering", func(t *testing.T) {
		t.Skip("TODO (b5): add a third dataset")
	})

	t.Run("user_2_automated_datasets", func(t *testing.T) {
		// miss piggy's collection should be empty. Kermit's collection is non-empty,
		// proving basic multi-tenancy
		if _, err := c.List(ctx, missPiggy.ID, params.ListAll); err == nil {
			t.Fatalf("listing without providing any keyIDs should error: %q", err)
		}

		muppetTweetsInitID := "muppetTweetsInitID"
		muppetTweetsName := "muppet_tweets"

		t.Skip("TODO (b5): need user-scoped events to make this work")

		// // simulate name initialization, normally emitted by logbook
		// mustPublish(ctx, t, bus, event.ETDatasetNameInit, event.DsChange{
		// 	InitID:     muppetTweetsInitID,
		// 	Username:   kermit.Peername,
		// 	ProfileID:  kermit.ID.String(),
		// 	PrettyName: muppetTweetsName,
		// 	Info: &dsref.VersionInfo{
		// 		InitID: muppetTweetsInitID,
		// 		Username: kermit,
		// 	},
		// })
		mustPublish(ctx, t, bus, event.ETRemoteClientPullDatasetCompleted, event.RemoteEvent{
			Ref: dsref.Ref{
				InitID:    muppetTweetsInitID,
				Username:  kermit.Peername,
				ProfileID: kermit.ID.String(),
			},
		})
		// simulate version creation, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetCommitChange, event.DsChange{
			InitID:   muppetTweetsInitID,
			TopIndex: 2,
			HeadRef:  "/mem/PathToMuppetTweetsVersionOne",
			// TODO (b5): real event includes a VersionInfo:
			// Info:     &info,
		})

		expect := []dsref.VersionInfo{
			{
				InitID:    muppetTweetsInitID,
				ProfileID: kermit.ID.String(),
				Username:  kermit.Peername,
				Name:      muppetTweetsName,
				// TopIndex:  2,
				// HeadRef:   "/mem/PathToMuppetTweetsVersionOne",
			},
		}
		assertCollectionList(ctx, t, missPiggy, params.ListAll, c, expect)

		// TODO (b5): simulate workflow creation, check that collection updates with
		// workflow ID
	})
}

func assertCollectionList(ctx context.Context, t *testing.T, p *profile.Profile, lp params.List, c collection.Set, expect []dsref.VersionInfo) {
	t.Helper()
	res, err := c.List(ctx, p.ID, lp)
	if err != nil {
		t.Fatalf("error listing items: %q", err)
	}
	expectItems(t, p.Peername, res, expect)
}

func expectItems(t *testing.T, username string, got, expect []dsref.VersionInfo) {
	t.Helper()
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Errorf("collection mismatch for user %q (-want +got):\n%s", username, diff)
	}
}

func mustPublish(ctx context.Context, t *testing.T, b event.Bus, typ event.Type, data interface{}) {
	t.Helper()
	if err := b.Publish(ctx, typ, data); err != nil {
		t.Fatalf("unepected error publishing event to bus: %s", err)
	}
}
