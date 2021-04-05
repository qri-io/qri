package spec

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/list"
	"github.com/qri-io/qri/profile"
	profiletest "github.com/qri-io/qri/profile/test"
)

type Constructor func(ctx context.Context, bus event.Bus) collection.Collection

func AssertCollectionSpec(t *testing.T, constructor Constructor) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bus := event.NewBus(ctx)
	c := constructor(ctx, bus)

	kermit := profiletest.GetProfile("kermit")
	missPiggy := profiletest.GetProfile("miss_piggy")

	t.Run("empty_unlimited_list", func(t *testing.T) {
		lp := list.Params{Offset: -1}
		res, err := c.List(ctx, profile.ID(""), lp)
		if err != nil {
			t.Fatalf("listing without providing any keyIDs should error: %q", err)
		}
		if res != nil {
			t.Errorf("expected nil items response when listing errors")
		}

		// new collection should return an empty list when listing a valid profileID
		assertCollectionList(ctx, t, kermit, lp, c, []collection.Item{})
	})

	t.Run("user_1_manual_datasets", func(t *testing.T) {
		muppetNamesInitID := "initID"
		muppetNamesName1 := "muppet_names"
		muppetNamesName2 := "muppet_names_and_ages"

		// simulate name initialization, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetNameInit, event.DsChange{
			InitID:     muppetNamesInitID,
			Username:   kermit.Peername,
			ProfileID:  string(kermit.ID),
			PrettyName: muppetNamesName1,
		})

		lp := list.Params{Offset: -1}
		expect := []collection.Item{
			{
				InitID:    muppetNamesInitID,
				ProfileID: string(kermit.ID),
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
			},
		}
		assertCollectionList(ctx, t, kermit, lp, c, expect)

		// simulate version creation, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetCommitChange, event.DsChange{
			InitID:   muppetNamesInitID,
			TopIndex: 2,
			HeadRef:  "/mem/PathToMuppetNamesVersionOne",
			// TODO (b5): real event includes a VersionInfo:
			// Info:     &info,
		})

		lp = list.Params{Offset: -1}
		expect = []collection.Item{
			{
				InitID:    muppetNamesInitID,
				ProfileID: string(kermit.ID),
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
				TopIndex:  2,
				HeadRef:   "/mem/PathToMuppetNamesVersionOne",
			},
		}
		assertCollectionList(ctx, t, kermit, lp, c, expect)

		// simulate dataset renaming, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetRename, event.DsChange{
			InitID:     muppetNamesInitID,
			PrettyName: muppetNamesName2,
		})

		lp = list.Params{Offset: -1}
		expect = []collection.Item{
			{
				InitID:    muppetNamesInitID,
				ProfileID: string(kermit.ID),
				Username:  kermit.Peername,
				Name:      muppetNamesName2,
				TopIndex:  2,
				HeadRef:   "/mem/PathToMuppetNamesVersionOne",
			},
		}
		assertCollectionList(ctx, t, kermit, lp, c, expect)

		// TODO (b5): create a second dataset, use different timestamps for both,
		// assert default ordering of datasets
	})

	t.Run("user_1_ordering_and_filtering", func(t *testing.T) {
		// TODO (b5): add a third dataset
	})

	t.Run("user_2_automated_datasets", func(t *testing.T) {
		lp := list.Params{Offset: -1}
		// miss piggy's collection should be empty. Kermit's collection is non-empty,
		// proving basic multi-tenancy
		assertCollectionList(ctx, t, missPiggy, lp, c, []collection.Item{})

		muppetTweetsInitID := "muppetTweetsInitID"
		muppetTweetsName := "muppet_tweets"

		// simulate name initialization, normally emitted by logbook
		mustPublish(ctx, t, bus, event.ETDatasetNameInit, event.DsChange{
			InitID:     muppetTweetsInitID,
			Username:   kermit.Peername,
			ProfileID:  string(kermit.ID),
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

		lp = list.Params{Offset: -1}
		expect := []collection.Item{
			{
				InitID:    muppetTweetsInitID,
				ProfileID: string(kermit.ID),
				Username:  kermit.Peername,
				Name:      muppetTweetsName,
				TopIndex:  2,
				HeadRef:   "/mem/PathToMuppetTweetsVersionOne",
			},
		}
		assertCollectionList(ctx, t, missPiggy, lp, c, expect)

		// TODO (b5): simulate workflow creation, check that collection updates with
		// workflow ID
	})
}

func assertCollectionList(ctx context.Context, t *testing.T, p *profile.Profile, lp list.Params, c collection.Collection, expect []collection.Item) {
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
