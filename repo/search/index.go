package search

import (
	"encoding/json"
	"github.com/ipfs/go-datastore"
	"log"
	"time"

	"github.com/qri-io/bleve"
	"github.com/qri-io/bleve/analysis/lang/en"
	//_ "github.com/qri-io/bleve/config"
	"github.com/qri-io/bleve/mapping"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

// IndexableMetadata specifies the subset of fields we want to keep from
// a dataset's metadata file to be used in the bleveindex
// ExternalScore and internalScore are placeholders for future use.
type IndexableMetadata struct {
	Category      string `json:"category"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Kind          string `json:"kind"`
	ExternalScore int    `json:"externalScore"`
	internalScore int
}

// NewIndexableMetadataStruct sets the default variable used to identify document type to 'table'
func NewIndexableMetadataStruct() *IndexableMetadata {
	return &IndexableMetadata{Kind: "table"}
}

// MapValues converts the IndexableMetadata back to type map[string]interface{}
func (imd *IndexableMetadata) MapValues() map[string]interface{} {
	return map[string]interface{}{
		"category":      imd.Category,
		"title":         imd.Title,
		"description":   imd.Description,
		"kind":          imd.Kind,
		"externalScore": imd.ExternalScore,
		"internalScore": imd.internalScore,
	}
}

var (
	// batch size for indexing
	batchSize = 100
)

// Index is an index of search data
type Index bleve.Index

// LoadIndex loads the search index
func LoadIndex(indexPath string) (Index, error) {
	// open the index
	repoIndex, err := bleve.Open(indexPath)
	if err == bleve.ErrorIndexPathDoesNotExist {
		log.Printf("Creating new index...")
		//create a mapping
		indexMapping, err := buildIndexMapping()
		if err != nil {
			return nil, err
		}

		repoIndex, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, err
		}

	} else if err != nil {
		return nil, err
	} else {
		// log.Printf("Opening existing index...")
	}

	return repoIndex, nil
}

func buildIndexMapping() (mapping.IndexMapping, error) {
	// a generic reusable mapping for english text
	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = en.AnalyzerName

	// a generic reusable mapping for things we want to ignore - not in use
	// dontStoreMeFieldMapping := bleve.NewTextFieldMapping()
	// dontStoreMeFieldMapping.Store = false
	// dontStoreMeFieldMapping.IncludeInAll = false
	// dontStoreMeFieldMapping.IncludeTermVectors = false
	// dontStoreMeFieldMapping.Index = false

	datasetMapping := bleve.NewDocumentMapping()
	//mappings for fields we want to index
	datasetMapping.AddFieldMappingsAt("title", englishTextFieldMapping)
	datasetMapping.AddFieldMappingsAt("description", englishTextFieldMapping)
	datasetMapping.AddFieldMappingsAt("category", englishTextFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("table", datasetMapping)
	indexMapping.TypeField = "kind"
	indexMapping.DefaultAnalyzer = "en"

	return indexMapping, nil
}

// IndexRepo calculates an index for a given repository
func IndexRepo(r repo.Repo, i bleve.Index) error {
	refs, err := r.Namespace(-1, 0)
	if err != nil {
		return err
	}
	return indexDatasetRefs(r.Store(), i, refs)
}

func indexDatasetRefs(store cafs.Filestore, i bleve.Index, refs []*repo.DatasetRef) error {
	log.Printf("Indexing...")
	count := 0
	startTime := time.Now()
	batch := i.NewBatch()
	batchCount := 0
	for _, ref := range refs {
		ds, err := dsfs.LoadDataset(store, datastore.NewKey(ref.Path))
		if err != nil {
			log.Printf("error loading dataset: %s", err.Error())
			continue
		}
		//remove extra fields
		data, err := json.Marshal(ds)
		if err != nil {
			log.Printf("error marshalling dataset: %s", err.Error())
			//continue
			return err
		}
		leanMetadata := NewIndexableMetadataStruct()
		json.Unmarshal(data, leanMetadata)

		batch.Index(ref.Path, leanMetadata.MapValues())
		batchCount++

		if batchCount >= batchSize {
			err = i.Batch(batch)
			if err != nil {
				return err
			}
			batch = i.NewBatch()
			batchCount = 0
		}
		count++
		// if count%1000 == 0 {
		// 	indexDuration := time.Since(startTime)
		// 	indexDurationSeconds := float64(indexDuration) / float64(time.Second)
		// 	timePerDoc := float64(indexDuration) / float64(count)
		// 	log.Printf("Indexed %d documents, in %.2fs average %2fms/doc)", count, indexDurationSeconds, timePerDoc/float64(time.Millisecond))
		// }
		// }
	}
	//flush the last batch
	if batchCount > 0 {
		err := i.Batch(batch)
		if err != nil {
			log.Fatal(err)
		}
	}
	indexDuration := time.Since(startTime)
	indexDurationSeconds := float64(indexDuration) / float64(time.Second)
	timePerDoc := float64(indexDuration) / float64(count)
	log.Printf("Indexed %d documents, in %.2fs (average %.2fms/doc)", count, indexDurationSeconds, timePerDoc/float64(time.Millisecond))
	return nil
}
