package dsfs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	cid "github.com/ipfs/go-cid"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/qri-io/dataset"
	"github.com/qri-io/dataset/dsviz"
	"github.com/qri-io/dataset/validate"
	"github.com/qri-io/qfs"
	"github.com/qri-io/qri/dsref"
	"github.com/qri-io/qri/event"
)

// number of entries to per batch when processing body data in WriteDataset
const batchSize = 5000

var (
	// BodySizeSmallEnoughToDiff sets how small a body must be to generate a message from it
	BodySizeSmallEnoughToDiff = 20000000 // 20M or less is small
	// OpenFileTimeoutDuration determines the maximium amount of time to wait for
	// a Filestore to open a file. Some filestores (like IPFS) fallback to a
	// network request when it can't find a file locally. Setting a short timeout
	// prevents waiting for a slow network response, at the expense of leaving
	// files unresolved.
	// TODO (b5) - allow -1 duration as a sentinel value for no timeout
	OpenFileTimeoutDuration = time.Millisecond * 700
)

// If a user has a dataset larger than the above limit, then instead of diffing we compare the
// checksum against the previous version. We should make this algorithm agree with how `status`
// works.
// See issue: https://github.com/qri-io/qri/issues/1150

// SaveSwitches represents options for saving a dataset
type SaveSwitches struct {
	// Use a custom timestamp, defaults to time.Now if unset
	Time time.Time
	// Replace is whether the save is a full replacement or a set of patches to previous
	Replace bool
	// Pin is whether the dataset should be pinned
	Pin bool
	// ConvertFormatToPrev is whether the body should be converted to match the previous format
	ConvertFormatToPrev bool
	// ForceIfNoChanges is whether the save should be forced even if no changes are detected
	ForceIfNoChanges bool
	// ShouldRender is deprecated, controls whether viz should be rendered
	ShouldRender bool
	// NewName is whether a new dataset should be created, guaranteeing there's no previous version
	NewName bool
	// FileHint is a hint for what file is used for creating this dataset
	FileHint string
	// Drop is a string of components to remove before saving
	Drop string
	// parsed drop string into list of components
	dropRevs []*dsref.Rev

	// action to take when calculating commit messages
	// bodyAction is set by computeFieldsFile to feed data to the commit component
	// write. A bit of a hack, but it works.
	bodyAct BodyAction
}

// CreateDataset writes a dataset to a provided store.
// Store is where we're going to store the data
// Dataset to be saved
// Prev is the previous version or nil if there isn't one
// Pk is the private key for cryptographically signing
// Sw is switches that control how the save happens
// Returns the immutable path if no error
func CreateDataset(
	ctx context.Context,
	source qfs.Filesystem,
	destination qfs.Filesystem,
	pub event.Publisher,
	ds *dataset.Dataset,
	prev *dataset.Dataset,
	pk crypto.PrivKey,
	sw SaveSwitches,
) (string, error) {
	if pk == nil {
		return "", fmt.Errorf("private key is required to create a dataset")
	}

	if err := DerefDataset(ctx, source, ds); err != nil {
		log.Debugf("dereferencing dataset components: %s", err)
		return "", err
	}
	if err := validate.Dataset(ds); err != nil {
		log.Debug(err.Error())
		return "", err
	}
	log.Debugw("CreateDataset", "ds.Peername", ds.Peername, "ds.Name", ds.Name, "dest", destination.Type())

	if prev != nil && !prev.IsEmpty() {
		log.Debugw("dereferencing previous dataset", "prevPath", prev.Path)
		if err := DerefDataset(ctx, source, prev); err != nil {
			log.Debug(err.Error())
			return "", err
		}
		if err := validate.Dataset(prev); err != nil {
			log.Debug(err.Error())
			return "", err
		}
	}

	peername := ds.Peername
	name := ds.Name

	go func() {
		evtErr := pub.Publish(ctx, event.ETDatasetSaveStarted, event.DsSaveEvent{
			Username:   peername,
			Name:       name,
			Message:    "save started",
			Completion: 0,
		})
		if evtErr != nil {
			log.Debugw("ignored error while publishing save start event", "evtErr", evtErr)
		}
	}()

	path, err := WriteDataset(ctx, source, destination, prev, ds, pub, pk, sw)
	if err != nil {
		log.Debug(err.Error())
		if evtErr := pub.Publish(ctx, event.ETDatasetSaveCompleted, event.DsSaveEvent{
			Username:   peername,
			Name:       name,
			Error:      err,
			Completion: 1.0,
		}); evtErr != nil {
			log.Debugw("ignored error while publishing save completed", "evtErr", evtErr)
		}
		return "", err
	}

	// TODO (b5) - many codepaths that call this function use the `ds` arg after saving
	// we need to dereference here so fields are set, but this is overkill if
	// the caller doesn't use the ds arg afterward
	// might make sense to have a wrapper function that writes and loads on success
	if err := DerefDataset(ctx, destination, ds); err != nil {
		if evtErr := pub.Publish(ctx, event.ETDatasetSaveCompleted, event.DsSaveEvent{
			Username:   peername,
			Name:       name,
			Error:      err,
			Completion: 1.0,
		}); evtErr != nil {
			log.Debugw("ignored error while publishing save completed", "evtErr", evtErr)
		}
		return path, err
	}

	return path, pub.Publish(ctx, event.ETDatasetSaveCompleted, event.DsSaveEvent{
		Username:   peername,
		Name:       name,
		Message:    "dataset saved",
		Path:       path,
		Completion: 1.0,
	})
}

// WriteDataset persists a datasets to a destination filesystem
func WriteDataset(
	ctx context.Context,
	src qfs.Filesystem,
	dst qfs.Filesystem,
	prev *dataset.Dataset,
	ds *dataset.Dataset,
	publisher event.Publisher,
	pk crypto.PrivKey,
	sw SaveSwitches,
) (string, error) {
	dstStore, ok := dst.(qfs.MerkleDagStore)
	if !ok {
		return "", fmt.Errorf("destination must be a MerkleDagStore")
	}

	if ds.Commit != nil {
		// assign timestamp early. saving process on large files can take many minutes
		// and we want to mark commit creation closer to when the user submitted the
		// creation request
		if ds.Commit.Timestamp.IsZero() {
			ds.Commit.Timestamp = Timestamp()
		} else {
			ds.Commit.Timestamp = ds.Commit.Timestamp.In(time.UTC)
		}
	}

	if ds.Stats != nil {
		ds.Stats = nil
	}

	revs, err := dsref.ParseRevs(sw.Drop)
	if err != nil {
		return "", err
	}
	sw.dropRevs = revs

	added := qfs.NewLinks()

	// the call order of these functions is important, funcs later in the slice
	// may rely on writeFiles fields set by eariler functions
	writeFuncs := []writeComponentFunc{
		bodyFileFunc(ctx, pk, publisher),      // no deps
		metadataFile,                          // no deps
		transformFile,                         // no deps
		structureFile,                         // requires bdoy if it exists
		statsFile,                             // requires body, structure if they exist
		readmeFile,                            // no deps
		vizFilesAddFunc(ctx, sw),              // requires body, meta, transform, structure, stats, readme if they exist
		commitFileAddFunc(ctx, pk, publisher), // requires meta, transform, body, structure, stats, readme, vizScript, vizRendered if they exist
		writeDatasetFile,                      // requires all other components
	}

	for _, fileFunc := range writeFuncs {
		if err := fileFunc(src, dstStore, prev, ds, added, &sw); err != nil {
			if errors.Is(errNoComponent, err) {
				continue
			}
			return "", err
		}
	}

	// add root node
	res, err := dstStore.PutNode(added)
	if err != nil {
		return "", err
	}
	return fsPathFromCID(dstStore, res.Cid), nil
}

// writeComponentFunc is a function that writes a component to a merkleDagStore
// it accepts a set of named links that have already been added
// write component funcs are expected to write a link to "added" on successful
// write
type writeComponentFunc func(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) (err error)

var errNoComponent = errors.New("no component")

func bodyFileFunc(ctx context.Context, pk crypto.PrivKey, publisher event.Publisher) writeComponentFunc {
	return func(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) error {
		if ds.BodyFile() == nil {
			if usePrevComponent(sw, "bd") && prev != nil && prev.BodyPath != "" {
				sw.bodyAct = BodySame
				// TODO (b5): need to validate that a potentially new structure will work
				if id, err := cidFromIPFSPath(prev.BodyPath); err == nil {
					added.Add(qfs.Link{Name: bodyFilename(prev), Cid: id, IsFile: true})
				}
			}
			return errNoComponent
		}

		sw.bodyAct = BodyDefault
		bodyFilename := bodyFilename(ds)
		cff, err := newComputeFieldsFile(ctx, publisher, pk, ds, prev, sw)
		if err != nil {
			return err
		}

		f, err := NewMemfileReader(bodyFilename, cff), nil
		if err != nil {
			return err
		}

		if err := writePackageFile(dst, f, added); err != nil {
			return err
		}
		if err := <-cff.(doneProcessingFile).DoneProcessing(); err != nil {
			return err
		}

		log.Debugw("setting calculated stats")
		ds.Stats, err = cff.(statsComponentFile).StatsComponent()
		return err
	}
}

func structureFile(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) error {
	if ds.Structure == nil {
		if usePrevComponent(sw, "st") && prev != nil && prev.Structure != nil {
			if id, err := cidFromIPFSPath(prev.Structure.Path); err == nil {
				log.Debugw("using previous structure", "path", prev.Structure.Path)
				added.Add(qfs.Link{Name: PackageFileStructure.String(), Cid: id, IsFile: true})
			}
		}
		return errNoComponent
	}

	ds.Structure.DropTransientValues()

	// if the destination filesystem is content-addressed, use the body
	// path as the checksum. Include path prefix to disambiguate which FS
	// generated the checksum
	if _, ok := dst.(qfs.CAFS); ok {
		if bodyLink := added.Get(ds.Structure.BodyFilename()); bodyLink != nil {
			ds.Structure.Checksum = fsPathFromCID(dst, bodyLink.Cid)
		}
	}

	f, err := JSONFile(PackageFileStructure.String(), ds.Structure)
	if err != nil {
		return err
	}
	return writePackageFile(dst, f, added)
}

func metadataFile(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) error {
	if ds.Meta == nil {
		if usePrevComponent(sw, "md") && prev != nil && prev.Meta != nil {
			if id, err := cidFromIPFSPath(prev.Meta.Path); err == nil {
				added.Add(qfs.Link{Name: PackageFileMeta.String(), Cid: id, IsFile: true})
			}
		}
		return errNoComponent
	}
	ds.Meta.DropTransientValues()
	f, err := JSONFile(PackageFileMeta.String(), ds.Meta)
	if err != nil {
		return err
	}
	return writePackageFile(dst, f, added)
}

func transformFile(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) error {
	if ds.Transform == nil {
		return errNoComponent
	}

	ds.Transform.DropTransientValues()
	// TODO (b5): this is validation logic, should happen before WriteDataset is
	// ever called.
	// all resources must be references
	for key, r := range ds.Transform.Resources {
		if r.Path == "" {
			return fmt.Errorf("transform resource %s requires a path to save", key)
		}
	}

	if tfsf := ds.Transform.ScriptFile(); tfsf != nil {
		if err := writePackageFile(dst, NewMemfileReader(transformScriptFilename, tfsf), added); err != nil {
			return err
		}
		link := added.Get(transformScriptFilename)
		ds.Transform.ScriptPath = fsPathFromCID(dst, link.Cid)
	}

	// // transform component is inlined into dataset
	// return errNoComponent
	f, err := JSONFile(PackageFileTransform.String(), ds.Transform)
	if err != nil {
		return err
	}

	return writePackageFile(dst, f, added)
}

func statsFile(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) error {
	if ds.Stats == nil {
		// if the body is unchanged and it's hash matches the prior, keep the stats component
		if usePrevComponent(sw, "bd") && usePrevComponent(sw, "sa") {
			if bdLnk := added.Get(bodyFilename(ds)); bdLnk != nil {
				if fsPathFromCID(dst, bdLnk.Cid) == prev.BodyPath && prev.Stats != nil && prev.Stats.Path != "" {
					if id, err := cidFromIPFSPath(prev.Stats.Path); err == nil {
						log.Debugw("body is unchanged, keeping stats component", "path", prev.Stats.Path)
						added.Add(qfs.Link{Name: PackageFileStats.String(), Cid: id})
					}
				}
			}
		}
		return errNoComponent
	}
	f, err := JSONFile(PackageFileStats.String(), ds.Stats)
	if err != nil {
		return err
	}
	return writePackageFile(dst, f, added)
}

func readmeFile(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) error {
	if ds.Readme == nil {
		if usePrevComponent(sw, "rm") && prev != nil && prev.Readme != nil {
			if id, err := cidFromIPFSPath(prev.Readme.Path); err == nil {
				added.Add(qfs.Link{Name: PackageFileReadme.String(), Cid: id, IsFile: true})
			}
		}
		return errNoComponent
	}

	ds.Readme.DropTransientValues()
	if rmsf := ds.Readme.ScriptFile(); rmsf != nil {
		f := NewMemfileReader(PackageFileReadmeScript.String(), rmsf)
		if err := writePackageFile(dst, f, added); err != nil {
			return err
		}
		ds.Readme.ScriptPath = fsPathFromCID(dst, added.Get(PackageFileReadmeScript.String()).Cid)
	}

	// readme is used for side-effects, component will be inlined into dataset component
	return errNoComponent
}

// TODO(b5): current construction makes it possible to provide both rendered
// file and script file externally, without checking that the rendered file is
// in fact the result of executing the script.
func vizFilesAddFunc(ctx context.Context, sw SaveSwitches) writeComponentFunc {
	return func(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) error {
		if ds.Viz == nil {
			if usePrevComponent(sw, "vz") && prev != nil && prev.Viz != nil {
				if id, err := cidFromIPFSPath(prev.Viz.Path); err == nil {
					added.Add(qfs.Link{Name: PackageFileViz.String(), Cid: id, IsFile: true})
				}
			}
			return errNoComponent
		}

		ds.Viz.DropTransientValues()
		vzfs := ds.Viz.ScriptFile()
		if vzfs != nil {
			if err := writePackageFile(dst, NewMemfileReader(PackageFileVizScript.String(), vzfs), added); err != nil {
				return err
			}
		}

		renderedF := ds.Viz.RenderedFile()
		if renderedF != nil {
			if err := writePackageFile(dst, NewMemfileReader(PackageFileRenderedViz.String(), renderedF), added); err != nil {
				return err
			}
		} else if vzfs != nil && sw.ShouldRender {
			renderDs := &dataset.Dataset{}
			renderDs.Assign(ds)

			if bfn := bodyFilename(ds); bfn != "" {
				if bodyLink := added.Get(bfn); bodyLink != nil {
					bf, err := dst.(qfs.Filesystem).Get(ctx, fsPathFromCID(dst, bodyLink.Cid))
					if err != nil {
						return err
					}
					renderDs.SetBodyFile(bf)
				}
			}

			if vizScriptLink := added.Get(PackageFileVizScript.String()); vizScriptLink != nil {
				sf, err := dst.(qfs.Filesystem).Get(ctx, fsPathFromCID(dst, vizScriptLink.Cid))
				if err != nil {
					return err
				}
				renderDs.Viz.SetScriptFile(sf)
			}

			result, err := dsviz.Render(renderDs)
			if err != nil {
				return err
			}
			if err := writePackageFile(dst, NewMemfileReader(PackageFileRenderedViz.String(), result), added); err != nil {
				return err
			}
		}

		// viz is used for side-effects, component will be inlined into dataset component
		return errNoComponent
	}
}

func writeDatasetFile(src qfs.Filesystem, dst qfs.MerkleDagStore, prev, ds *dataset.Dataset, added qfs.Links, sw *SaveSwitches) error {
	if added.Len() == 0 {
		return fmt.Errorf("cannot save empty dataset")
	}

	ds.DropTransientValues()
	updateScriptPaths(dst, ds, added)
	setComponentRefs(dst, ds, bodyFilename(ds), added)

	f, err := JSONFile(PackageFileDataset.String(), ds)
	if err != nil {
		return err
	}
	return writePackageFile(dst, f, added)
}

func updateScriptPaths(s qfs.MerkleDagStore, ds *dataset.Dataset, added qfs.Links) {
	for filename, link := range added.Map() {
		path := fsPathFromCID(s, link.Cid)
		switch filename {
		case PackageFileVizScript.String():
			ds.Viz.ScriptPath = path
		case PackageFileRenderedViz.String():
			ds.Viz.RenderedPath = path
		case PackageFileReadmeScript.String():
			ds.Readme.ScriptPath = path
		}
	}
}

func fsPathFromCID(s qfs.MerkleDagStore, id cid.Cid) string {
	fs := s.(qfs.Filesystem)
	return fmt.Sprintf("/%s/%s", fs.Type(), id.String())
}

func cidFromIPFSPath(path string) (cid.Cid, error) {
	if !strings.HasPrefix(path, "/ipfs/") {
		return cid.Cid{}, fmt.Errorf("cannot create link to path oustide of ipfs filesystem")
	}
	return cid.Parse(strings.TrimPrefix(path, "/ipfs/"))
}

func writePackageFile(s qfs.MerkleDagStore, f fs.File, added qfs.Links) error {
	fi, err := f.Stat()
	if err != nil {
		return err
	}

	res, err := s.PutFile(f)
	if err != nil {
		return err
	}

	added.Add(res.ToLink(fi.Name(), !fi.IsDir()))
	return nil
}

func bodyFilename(ds *dataset.Dataset) string {
	if ds.Structure == nil {
		return ""
	}
	return ds.Structure.BodyFilename()
}

func setComponentRefs(dst qfs.MerkleDagStore, ds *dataset.Dataset, bodyFilename string, added qfs.Links) {
	for filename, link := range added.Map() {
		switch filename {
		case bodyFilename:
			ds.BodyPath = fsPathFromCID(dst, link.Cid)
		case PackageFileCommit.String():
			ds.Commit = dataset.NewCommitRef(fsPathFromCID(dst, link.Cid))
		case PackageFileMeta.String():
			ds.Meta = dataset.NewMetaRef(fsPathFromCID(dst, link.Cid))
		case PackageFileViz.String():
			ds.Viz = dataset.NewVizRef(fsPathFromCID(dst, link.Cid))
		case PackageFileStats.String():
			ds.Stats = dataset.NewStatsRef(fsPathFromCID(dst, link.Cid))
		case PackageFileStructure.String():
			ds.Structure = dataset.NewStructureRef(fsPathFromCID(dst, link.Cid))
			// TODO(b5): bug!
			// case PackageFileTransform.String():
			// 	ds.Transform = dataset.NewTransformRef(fsPathFromCID(dst, link.Cid))
		}
	}
}

func usePrevComponent(sw *SaveSwitches, component string) bool {
	if sw.Replace {
		return false
	}
	for _, rev := range sw.dropRevs {
		if rev.Field == component {
			return false
		}
	}
	return true
}
