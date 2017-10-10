package search

import (
	"log"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/lang/en"
	"github.com/blevesearch/bleve/mapping"
	"github.com/qri-io/cafs"
	"github.com/qri-io/dataset/dsfs"
	"github.com/qri-io/qri/repo"
)

var (
	// batch size for indexing
	batchSize = 100
)

type Index bleve.Index

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
		log.Printf("Opening existing index...")
	}

	return repoIndex, nil
}

func buildIndexMapping() (mapping.IndexMapping, error) {
	// a generic reusable mapping for english text
	englishTextFieldMapping := bleve.NewTextFieldMapping()
	englishTextFieldMapping.Analyzer = en.AnalyzerName

	// a generic reusable mapping for things we want to ignore
	dontStoreMeFieldMapping := bleve.NewTextFieldMapping()
	dontStoreMeFieldMapping.Store = false
	dontStoreMeFieldMapping.IncludeInAll = false
	dontStoreMeFieldMapping.IncludeTermVectors = false
	dontStoreMeFieldMapping.Index = false

	datasetMapping := bleve.NewDocumentMapping()
	//mappings for things we want
	datasetMapping.AddFieldMappingsAt("title", englishTextFieldMapping)
	datasetMapping.AddFieldMappingsAt("description", englishTextFieldMapping)

	//things we don't want
	// datasetMapping.AddFieldMappingsAt("pgTitle", dontStoreMeFieldMapping)
	// datasetMapping.AddFieldMappingsAt("sectionTitle", dontStoreMeFieldMapping)
	// datasetMapping.AddFieldMappingsAt("numDataRows", dontStoreMeFieldMapping)
	// datasetMapping.AddFieldMappingsAt("tableCaption", dontStoreMeFieldMapping)
	// datasetMapping.AddFieldMappingsAt("_id", dontStoreMeFieldMapping)
	// datasetMapping.AddFieldMappingsAt("pgId", dontStoreMeFieldMapping)
	// datasetMapping.AddFieldMappingsAt("numCols", dontStoreMeFieldMapping)
	// datasetMapping.AddFieldMappingsAt("numHeaderRows", dontStoreMeFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("table", datasetMapping)
	indexMapping.TypeField = "type"
	indexMapping.DefaultAnalyzer = "en"

	return indexMapping, nil
}

func IndexRepo(store cafs.Filestore, r repo.Repo, i bleve.Index) error {
	refs, err := r.Namespace(-1, 0)
	if err != nil {
		return err
	}
	return indexDatasetRefs(store, i, refs)
}

func indexDatasetRefs(store cafs.Filestore, i bleve.Index, refs []*repo.DatasetRef) error {
	log.Printf("Indexing...")
	count := 0
	startTime := time.Now()
	batch := i.NewBatch()
	batchCount := 0
	for _, ref := range refs {
		ds, err := dsfs.LoadDataset(store, ref.Path)
		if err != nil {
			log.Printf("error loading dataset: %s", err.Error())
			continue
		}

		batch.Index(ref.Path.String(), ds)
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
