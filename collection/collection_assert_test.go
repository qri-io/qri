package collection_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/qri-io/qri/automation/workflow"
	"github.com/qri-io/qri/base/params"
	"github.com/qri-io/qri/collection"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
	"github.com/qri-io/qri/profile"
	profiletest "github.com/qri-io/qri/profile/test"
	"github.com/qri-io/qri/transform"
)

// Constructor is a function for creating collections, used by spec tests
type Constructor func(ctx context.Context) (collection.Set, error)

// AssertSetSpec defines expected behaviours for a Writable
// collection implementation
func AssertSetSpec(t *testing.T, constructor Constructor) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ec, err := constructor(ctx)
	if err != nil {
		t.Fatal(err)
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

	t.Run("add", func(t *testing.T) {
		badItems := []struct {
			problem string
			item    dsref.VersionInfo
		}{
			{"empty", dsref.VersionInfo{}},
			{"no InitID", dsref.VersionInfo{ProfileID: kermit.ID.Encode()}},
			{"no profileID", dsref.VersionInfo{InitID: "init_id"}},
			{"no name", dsref.VersionInfo{InitID: "init_id", ProfileID: kermit.ID.Encode()}},
		}

		for _, bad := range badItems {
			t.Run(fmt.Sprintf("bad_item_%s", bad.problem), func(t *testing.T) {
				if err := ec.Add(ctx, kermit.ID, bad.item); err == nil {
					t.Error("expected error, got nil")
				}
			})
		}

		err := ec.Add(ctx, kermit.ID,
			dsref.VersionInfo{
				ProfileID:  kermit.ID.Encode(),
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			dsref.VersionInfo{
				ProfileID:  kermit.ID.Encode(),
				InitID:     "muppet_names_and_ages_init_id",
				Username:   "kermit",
				Name:       "muppet_names_and_ages",
				CommitTime: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		)

		if err != nil {
			t.Fatalf("error adding items: %s", err)
		}

		err = ec.Add(ctx, missPiggy.ID,
			dsref.VersionInfo{
				ProfileID:  missPiggy.ID.Encode(),
				InitID:     "secret_muppet_friends_init_id",
				Username:   "miss_piggy",
				Name:       "secret_muppet_friends",
				CommitTime: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
			dsref.VersionInfo{
				ProfileID:  missPiggy.ID.Encode(),
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			dsref.VersionInfo{
				ProfileID:  missPiggy.ID.Encode(),
				InitID:     "famous_muppets_init_id",
				Username:   "famous_muppets",
				Name:       "famous_muppets",
				CommitTime: time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
			},
		)

		if err != nil {
			t.Fatalf("error adding items: %s", err)
		}
	})
	t.Run("list", func(t *testing.T) {
		assertCollectionList(ctx, t, kermit, params.ListAll, ec, []dsref.VersionInfo{
			{
				ProfileID:  kermit.ID.Encode(),
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ProfileID:  kermit.ID.Encode(),
				InitID:     "muppet_names_and_ages_init_id",
				Username:   "kermit",
				Name:       "muppet_names_and_ages",
				CommitTime: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		})

		assertCollectionList(ctx, t, missPiggy, params.ListAll, ec, []dsref.VersionInfo{
			{
				ProfileID:  missPiggy.ID.Encode(),
				InitID:     "famous_muppets_init_id",
				Username:   "famous_muppets",
				Name:       "famous_muppets",
				CommitTime: time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
			},
			{
				ProfileID:  missPiggy.ID.Encode(),
				InitID:     "muppet_names_init_id",
				Username:   "kermit",
				Name:       "muppet_names",
				CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ProfileID:  missPiggy.ID.Encode(),
				InitID:     "secret_muppet_friends_init_id",
				Username:   "miss_piggy",
				Name:       "secret_muppet_friends",
				CommitTime: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
			},
		})
	})

	t.Run("delete", func(t *testing.T) {
		badCases := []struct {
			reason    string
			profileID profile.ID
			ids       string
		}{
			{"missing ID", kermit.ID, "unknown"},
		}

		for _, bad := range badCases {
			t.Run(fmt.Sprintf("bad_case_%s", bad.reason), func(t *testing.T) {
				if err := ec.Delete(ctx, bad.profileID, bad.ids); err == nil {
					t.Errorf("expected bad case to error. got nil")
				}
			})
		}

		err := ec.Delete(ctx, missPiggy.ID, "famous_muppets_init_id")
		if err != nil {
			t.Errorf("unexpected error deleting items: %s", err)
		}
		err = ec.Delete(ctx, missPiggy.ID, "muppet_names_init_id")
		if err != nil {
			t.Errorf("unexpected error deleting items: %s", err)
		}

		err = ec.Delete(ctx, missPiggy.ID, "secret_muppet_friends_init_id")
		if err != nil {
			t.Errorf("unexpected error deleting items: %s", err)
		}

		assertCollectionList(ctx, t, missPiggy, params.ListAll, ec, []dsref.VersionInfo{})
	})

	t.Run("get", func(t *testing.T) {
		muppetDSInitID := "muppet_DS_init_id"
		expect := &dsref.VersionInfo{
			ProfileID:  kermit.ID.Encode(),
			InitID:     muppetDSInitID,
			Username:   "kermit",
			Name:       "muppet_names",
			CommitTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		err = ec.Add(ctx, kermit.ID, *expect)
		if err != nil {
			t.Fatalf("error adding items: %s", err)
		}

		got, err := ec.Get(ctx, kermit.ID, muppetDSInitID)
		if err != nil {
			t.Fatalf("error getting version info: %s", err)
		}
		if diff := cmp.Diff(expect, got); diff != "" {
			t.Errorf("collection get version info mismatch (-want +got):\n%s", diff)
		}

		got, err = ec.Get(ctx, kermit.ID, "bad_init_id")
		if !errors.Is(err, collection.ErrNotFound) {
			t.Errorf("error mismatch, expected %q, got %q", collection.ErrNotFound, err)
		}

		got, err = ec.Get(ctx, missPiggy.ID, muppetDSInitID)
		if !errors.Is(err, collection.ErrNotFound) {
			t.Errorf("error mismatch, expected %q, got %q", collection.ErrNotFound, err)
		}
	})

}

// AssertCollectionEventListenerSpec defines expected behaviours for collections
// that use event subscriptions to update state.
// Event Listener specs are optional.
func AssertCollectionEventListenerSpec(t *testing.T, constructor Constructor) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, err := constructor(ctx)
	if err != nil {
		t.Fatal(err)
	}

	bus := event.NewBus(ctx)
	_, err = collection.NewSetMaintainer(ctx, bus, s)
	if err != nil {
		t.Fatal(err)
	}

	kermit := profiletest.GetProfile("kermit")
	missPiggy := profiletest.GetProfile("miss_piggy")

	t.Run("empty_unlimited_list", func(t *testing.T) {
		_, err := s.List(ctx, profile.ID(""), params.ListAll)
		if err == nil {
			t.Fatalf("listing without providing any keyIDs should error: %q", err)
		}
	})

	t.Run("user_1_manual_datasets", func(t *testing.T) {
		muppetNamesInitID := "initID"
		muppetNamesName1 := "muppet_names"
		muppetNamesName2 := "muppet_names_and_ages"
		muppetNamesRunID1 := "muppet_names_run_id_1"
		muppetNamesRunID2 := "muppet_names_run_id_2"

		// initialize a dataset with the given name, initID, and profileID
		mustPublish(ctx, t, bus, event.ETDatasetNameInit, dsref.VersionInfo{
			InitID:    muppetNamesInitID,
			ProfileID: kermit.ID.Encode(),
			Username:  kermit.Peername,
			Name:      muppetNamesName1,
		})

		expect := []dsref.VersionInfo{
			{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID.Encode(),
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
			},
		}
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate dataset download
		mustPublish(ctx, t, bus, event.ETDatasetDownload, muppetNamesInitID)
		expect[0].DownloadCount = 1
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate a commit transform
		mustPublish(ctx, t, bus, event.ETTransformStart, event.TransformLifecycle{RunID: muppetNamesRunID1, Mode: transform.RMCommit, InitID: muppetNamesInitID})
		expect[0].RunCount = 1
		expect[0].RunID = muppetNamesRunID1
		expect[0].RunStatus = "running"
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate a "WriteTransformRun" in logbook
		// this might occur if a transform has errored or resulted in no changes
		mustPublish(ctx, t, bus, event.ETLogbookWriteRun, dsref.VersionInfo{InitID: muppetNamesInitID, RunID: muppetNamesRunID1, RunStatus: "unchanged", RunDuration: 1000})
		expect[0].RunStatus = "unchanged"
		expect[0].RunDuration = 1000

		// simulate version creation with no transform
		mustPublish(ctx, t, bus, event.ETLogbookWriteCommit, dsref.VersionInfo{
			InitID:      muppetNamesInitID,
			ProfileID:   kermit.ID.Encode(),
			Path:        "/mem/PathToMuppetNamesVersionOne",
			Username:    kermit.Peername,
			Name:        muppetNamesName1,
			CommitCount: 2,
			BodySize:    20,
		})

		expect[0].CommitCount = 2
		expect[0].BodySize = 20
		expect[0].Path = "/mem/PathToMuppetNamesVersionOne"
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate version creation with a transform
		mustPublish(ctx, t, bus, event.ETLogbookWriteCommit, dsref.VersionInfo{
			InitID:      muppetNamesInitID,
			ProfileID:   kermit.ID.Encode(),
			Path:        "/mem/PathToMuppetNamesVersionTwo",
			Username:    kermit.Peername,
			Name:        muppetNamesName1,
			CommitCount: 3,
			BodySize:    25,
			RunID:       muppetNamesRunID2,
			RunStatus:   "success",
			RunDuration: 2000,
		})
		expect[0].Path = "/mem/PathToMuppetNamesVersionTwo"
		expect[0].CommitCount = 3
		expect[0].BodySize = 25
		expect[0].RunID = muppetNamesRunID2
		expect[0].RunStatus = "success"
		expect[0].RunDuration = 2000
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate dataset being renamed
		mustPublish(ctx, t, bus, event.ETDatasetRename, event.DsRename{
			InitID:  muppetNamesInitID,
			OldName: muppetNamesName1,
			NewName: muppetNamesName2,
		})

		expect[0].Name = muppetNamesName2
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// dataset deleted using a scope associated with the owning profile
		{
			scopedCtx := profile.AddIDToContext(ctx, kermit.ID.Encode())
			mustPublish(scopedCtx, t, bus, event.ETDatasetDeleteAll, muppetNamesInitID)
		}

		expect = []dsref.VersionInfo{}
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate another initialization
		mustPublish(ctx, t, bus, event.ETDatasetNameInit, dsref.VersionInfo{
			InitID:    muppetNamesInitID,
			ProfileID: kermit.ID.Encode(),
			Username:  kermit.Peername,
			Name:      muppetNamesName1,
		})

		expect = []dsref.VersionInfo{
			{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID.Encode(),
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
			},
		}
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// deletion event without the owning profile has no effect
		mustPublish(ctx, t, bus, event.ETDatasetDeleteAll, muppetNamesInitID)
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// dataset deleted using a scope associated with the owning profile
		{
			scopedCtx := profile.AddIDToContext(ctx, kermit.ID.Encode())
			mustPublish(scopedCtx, t, bus, event.ETDatasetDeleteAll, muppetNamesInitID)
		}

		expect = []dsref.VersionInfo{}
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// TODO (b5): create a second dataset, use different timestamps for both,
		// assert default ordering of datasets
	})

	t.Run("user_1_remote_actions", func(t *testing.T) {
		muppetNamesInitID := "initID"
		muppetNamesName1 := "muppet_names"

		// initialize a dataset with the given name, initID, and profileID
		mustPublish(ctx, t, bus, event.ETDatasetNameInit, dsref.VersionInfo{
			InitID:    muppetNamesInitID,
			ProfileID: kermit.ID.Encode(),
			Username:  kermit.Peername,
			Name:      muppetNamesName1,
		})

		expect := []dsref.VersionInfo{
			{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID.Encode(),
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
			},
		}
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate that someone has followed the dataset
		mustPublish(ctx, t, bus, event.ETRemoteDatasetFollowed, muppetNamesInitID)
		expect[0].FollowerCount = 1
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate that someone has unfollowed the dataset
		mustPublish(ctx, t, bus, event.ETRemoteDatasetUnfollowed, muppetNamesInitID)
		expect[0].FollowerCount = 0
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate that someone has opened an issue on the dataset
		mustPublish(ctx, t, bus, event.ETRemoteDatasetIssueOpened, muppetNamesInitID)
		expect[0].OpenIssueCount = 1
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// simulate that someone has closed an issue on the dataset
		mustPublish(ctx, t, bus, event.ETRemoteDatasetIssueClosed, muppetNamesInitID)
		expect[0].OpenIssueCount = 0
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// dataset deleted using a scope associated with the owning profile
		{
			scopedCtx := profile.AddIDToContext(ctx, kermit.ID.Encode())
			mustPublish(scopedCtx, t, bus, event.ETDatasetDeleteAll, muppetNamesInitID)
		}

		expect = []dsref.VersionInfo{}
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)
	})

	t.Run("user_1_ordering_and_filtering", func(t *testing.T) {
		t.Skip("TODO (b5): add a third dataset")
	})

	t.Run("user_2_automation_datasets", func(t *testing.T) {
		// miss piggy's collection should be empty. Kermit's collection is non-empty,
		// proving basic multi-tenancy
		if _, err := s.List(ctx, missPiggy.ID, params.ListAll); err == nil {
			t.Fatalf("listing without providing any keyIDs should error: %q", err)
		}

		// automated dataset tests
		missPiggyDatasetInitID := "missPiggyDatasetInitID"
		missPiggyDatasetName := "miss_piggy_dataset"

		mustPublish(ctx, t, bus, event.ETDatasetNameInit, dsref.VersionInfo{
			InitID:      missPiggyDatasetInitID,
			CommitCount: 1,
			Path:        "/mem/PathToMissPiggyDatasetVersionOne",
			ProfileID:   missPiggy.ID.Encode(),
			Username:    missPiggy.Peername,
			Name:        missPiggyDatasetName,
		})

		expect := []dsref.VersionInfo{
			{
				InitID:      missPiggyDatasetInitID,
				ProfileID:   missPiggy.ID.Encode(),
				Username:    missPiggy.Peername,
				Name:        missPiggyDatasetName,
				CommitCount: 1,
				Path:        "/mem/PathToMissPiggyDatasetVersionOne",
			},
		}
		assertCollectionList(ctx, t, missPiggy, params.ListAll, s, expect)

		// simulate workflow creation, check that collection updates with
		// workflow ID
		wf := workflow.Workflow{
			InitID:  missPiggyDatasetInitID,
			OwnerID: missPiggy.ID,
			ID:      "workflow_id",
		}
		mustPublish(ctx, t, bus, event.ETAutomationWorkflowCreated, wf)

		expect[0].WorkflowID = "workflow_id"
		assertCollectionList(ctx, t, missPiggy, params.ListAll, s, expect)

		mustPublish(ctx, t, bus, event.ETLogbookWriteCommit, dsref.VersionInfo{
			InitID:      missPiggyDatasetInitID,
			CommitCount: 2,
			Path:        "/mem/PathToMissPiggyDatasetVersionTwo",
			ProfileID:   missPiggy.ID.Encode(),
			Username:    missPiggy.Peername,
			Name:        missPiggyDatasetName,
		})
		expect[0].CommitCount = 2
		expect[0].Path = "/mem/PathToMissPiggyDatasetVersionTwo"
		assertCollectionList(ctx, t, missPiggy, params.ListAll, s, expect)

		// simulate workflow removal, check that the collection removes workflowID
		mustPublish(ctx, t, bus, event.ETAutomationWorkflowRemoved, wf)
		expect[0].WorkflowID = ""
		assertCollectionList(ctx, t, missPiggy, params.ListAll, s, expect)

		// dataset deleted using a scope associated with the owning profile
		{
			scopedCtx := profile.AddIDToContext(ctx, missPiggy.ID.Encode())
			mustPublish(scopedCtx, t, bus, event.ETDatasetDeleteAll, missPiggyDatasetInitID)
		}
		expect = []dsref.VersionInfo{}
		assertCollectionList(ctx, t, missPiggy, params.ListAll, s, expect)
	})

	t.Run("user_3_pull_dataset", func(t *testing.T) {
		muppetNamesInitID := "initID"
		muppetNamesName1 := "muppet_names"

		// no user profile in the context, so dataset pull does nothing
		mustPublish(ctx, t, bus, event.ETDatasetPulled, dsref.VersionInfo{
			InitID:    muppetNamesInitID,
			ProfileID: kermit.ID.Encode(),
			Username:  kermit.Peername,
			Name:      muppetNamesName1,
		})
		expect := []dsref.VersionInfo{}
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// add a user profile to the scoped context
		{
			scopedCtx := profile.AddIDToContext(ctx, kermit.ID.Encode())
			// pull again which will work this time
			mustPublish(scopedCtx, t, bus, event.ETDatasetPulled, dsref.VersionInfo{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID.Encode(),
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
			})
		}

		// now collection has the expected info
		expect = []dsref.VersionInfo{
			{
				InitID:    muppetNamesInitID,
				ProfileID: kermit.ID.Encode(),
				Username:  kermit.Peername,
				Name:      muppetNamesName1,
			},
		}
		assertCollectionList(ctx, t, kermit, params.ListAll, s, expect)

		// another user's collection should not be affected
		expect = []dsref.VersionInfo{}
		assertCollectionList(ctx, t, missPiggy, params.ListAll, s, expect)
	})
}

func assertCollectionList(ctx context.Context, t *testing.T, p *profile.Profile, lp params.List, s collection.Set, expect []dsref.VersionInfo) {
	t.Helper()
	res, err := s.List(ctx, p.ID, lp)
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
