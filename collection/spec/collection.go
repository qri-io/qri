package spec

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/params"
	"github.com/qri-io/qri/profile"
	profiletest "github.com/qri-io/qri/profile/test"
)

// Constructor is a function for creating collections, used by spec tests
type Constructor func(ctx context.Context, bus event.Bus) collection.Collection

// AssertWritableCollectionSpec defines expected behaviours for a Writable
// collection implementation
func AssertWritableCollectionSpec(t *testing.T, constructor Constructor) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus := event.NewBus(ctx)

	c := constructor(ctx, bus)
	ec, ok := c.(collection.Writable)
	if !ok {
		t.Fatal("construtor did not return an editable collection")
	}

	kermit := profiletest.GetProfile("kermit")
	missPiggy := profiletest.GetProfile("miss_piggy")

	t.Run("empty_list", func(t *testing.T) {
		res, err := ec.List(ctx, missPiggy.ID, params.ListAll)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
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
			item    collection.Item
		}{
			{"empty", collection.Item{}},
			{"no InitID", collection.Item{ProfileID: kermit.ID}},
			{"no profileID", collection.Item{InitID: "init_id"}},
			{"no name", collection.Item{InitID: "init_id", ProfileID: kermit.ID}},
		}

		for _, bad := range badItems {
			t.Run(fmt.Sprintf("bad_item_%s", bad.problem), func(t *testing.T) {
				if err := ec.Put(ctx, kermit.ID, bad.item); err == nil {
					t.Errorf("expected error, got nil", bad.problem)
				}
			})
		}

		err := ec.Put(ctx, kermit.ID,
			collection.Item{
				ProfileID:  kermit.ID,
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			collection.Item{
				ProfileID:  kermit.ID,
				InitID:     "muppet_names_and_ages_init_id",
				Username:   "kermit",
				Name:       "muppet_names_and_ages",
				CommitTime: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		)

		if err != nil {
			t.Fatal("error adding items: %s", err)
		}

		err = ec.Put(ctx, missPiggy.ID,
			collection.Item{
				ProfileID:  missPiggy.ID,
				InitID:     "secret_muppet_friends_init_id",
				Username:   "miss_piggy",
				Name:       "secret_muppet_friends",
				CommitTime: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
			collection.Item{
				ProfileID:  missPiggy.ID,
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			collection.Item{
				ProfileID:  missPiggy.ID,
				InitID:     "famous_muppets_init_id",
				Username:   "famous_muppets",
				Name:       "famous_muppets",
				CommitTime: time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
			})

		if err != nil {
			t.Fatal("error adding items: %s", err)
		}
	})

	t.Run("list", func(t *testing.T) {
		assertCollectionList(ctx, t, kermit, params.ListAll, ec, []collection.Item{
			{
				ProfileID:  kermit.ID,
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ProfileID:  kermit.ID,
				InitID:     "muppet_names_and_ages_init_id",
				Username:   "kermit",
				Name:       "muppet_names_and_ages",
				CommitTime: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		})

		assertCollectionList(ctx, t, missPiggy, params.ListAll, ec, []collection.Item{
			{
				ProfileID:  missPiggy.ID,
				InitID:     "famous_muppets_init_id",
				Username:   "famous_muppets",
				Name:       "famous_muppets",
				CommitTime: time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
			},
			{
				ProfileID:  missPiggy.ID,
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ProfileID:  missPiggy.ID,
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
				if err := ec.Delete(ctx, bad.profileID, bad.ids...); err != nil {
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

		assertCollectionList(ctx, t, missPiggy, params.ListAll, ec, []collection.Item{})
	})
}

// AssertCollectionEventSpec defines expected behaviours for collection
// implementations & subscribed events
// TODO(b5): this is a planned spec, has no implementations
func AssertCollectionEventSpec(t *testing.T, constructor Constructor) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus := event.NewBus(ctx)
	c := constructor(ctx, bus)

	kermit := profiletest.GetProfile("kermit")
	missPiggy := profiletest.GetProfile("miss_piggy")

	t.Run("empty_unlimited_list", func(t *testing.T) {
		res, err := c.List(ctx, profile.ID(""), params.ListAll)
		if err != nil {
			t.Fatalf("listing without providing any keyIDs should error: %q", err)
		}
		if res != nil {
			t.Errorf("expected nil items response when listing errors")
		}

		// new collection should return an empty list when listing a valid profileID
		assertCollectionList(ctx, t, kermit, params.ListAll, c, []collection.Item{})
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
		})

		expect := []collection.Item{
			{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID,
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
			},
		}
		assertCollectionList(ctx, t, kermit, params.ListAll, c, expect)

		// simulate version creation, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetCommitChange, event.DsChange{
			InitID:   muppetNamesInitID,
			TopIndex: 2,
			HeadRef:  "/mem/PathToMuppetNamesVersionOne",
			// TODO (b5): real event includes a VersionInfo:
			// Info:     &info,
		})

		expect = []collection.Item{
			{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID,
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
				TopIndex:  2,
				HeadRef:   "/mem/PathToMuppetNamesVersionOne",
			},
		}
		assertCollectionList(ctx, t, kermit, params.ListAll, c, expect)

		// simulate dataset renaming, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetRename, event.DsChange{
			InitID:     muppetNamesInitID,
			PrettyName: muppetNamesName2,
		})

		expect = []collection.Item{
			{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID,
				Username:  kermit.Peername,
				Name:      muppetNamesName2,
				TopIndex:  2,
				HeadRef:   "/mem/PathToMuppetNamesVersionOne",
			},
		}
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
		assertCollectionList(ctx, t, missPiggy, params.ListAll, c, []collection.Item{})

		muppetTweetsInitID := "muppetTweetsInitID"
		muppetTweetsName := "muppet_tweets"

		// simulate name initialization, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetNameInit, event.DsChange{
			InitID:     muppetTweetsInitID,
			Username:   kermit.Peername,
			ProfileID:  kermit.ID.String(),
			PrettyName: muppetTweetsName,
		})
		// simulate version creation, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetCommitChange, event.DsChange{
			InitID:   muppetTweetsInitID,
			TopIndex: 2,
			HeadRef:  "/mem/PathToMuppetTweetsVersionOne",
			// TODO (b5): real event includes a VersionInfo:
			// Info:     &info,
		})

		expect := []collection.Item{
			{
				InitID:    muppetTweetsInitID,
				ProfileID: kermit.ID,
				Username:  kermit.Peername,
				Name:      muppetTweetsName,
				TopIndex:  2,
				HeadRef:   "/mem/PathToMuppetTweetsVersionOne",
			},
		}
		assertCollectionList(ctx, t, missPiggy, params.ListAll, c, expect)

		// TODO (b5): simulate workflow creation, check that collection updates with
		// workflow ID
	})
}

func assertCollectionList(ctx context.Context, t *testing.T, p *profile.Profile, lp params.List, c collection.Collection, expect []collection.Item) {
	t.Helper()
	res, err := c.List(ctx, p.ID, lp)
	if err != nil {
		t.Fatalf("error listing items: %q", err)
	}
	expectItems(t, p.Peername, res, expect)
}

func expectItems(t *testing.T, username string, got, expect []collection.Item) {
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
