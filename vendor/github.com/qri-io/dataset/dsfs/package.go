package dsfs

type PackageFile int

const (
	PackageFileUnknown PackageFile = iota
	PackageFileDataset
	PackageFileStructure
	PackageFileAbstractStructure
	PackageFileResources
	PackageFileCommitMsg
	PackageFileQuery
)

var filenames = map[PackageFile]string{
	PackageFileUnknown:           "",
	PackageFileDataset:           "dataset.json",
	PackageFileStructure:         "structure.json",
	PackageFileAbstractStructure: "abstract_structure.json",
	PackageFileResources:         "resources",
	PackageFileCommitMsg:         "commit.json",
	PackageFileQuery:             "query.json",
}

func (p PackageFile) String() string {
	return p.Filename()
}

func (p PackageFile) Filename() string {
	return filenames[p]
}
