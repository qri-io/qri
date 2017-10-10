package search

// import (
// 	"encoding/json"
// 	"flag"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"net/http"
// 	"os"
// 	"path/filepath"
// 	"runtime"
// 	"runtime/pprof"
// 	"strings"
// 	"time"

// 	"github.com/blevesearch/bleve"
// 	_ "github.com/blevesearch/bleve/config"
// 	bleveHttp "github.com/blevesearch/bleve/http"
// 	"github.com/gorilla/mux"
// )

// var batchSize = flag.Int("batchSize", 100, "batch size for indexing")
// var bindAddr = flag.String("addr", ":8094", "http listen address")
// var jsonDir = flag.String("jsonDir", "/Volumes/LaCie/wikitables/bleve/", "json directory")
// var indexPath = flag.String("index", "qri-table-index.bleve", "index path")
// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
// var memprofile = flag.String("memprofile", "", "write mem profile to file")

// func printConfig() {
// 	fmt.Printf("batchSize: %d\n", *batchSize)
// 	fmt.Printf("port #: %s\n", *bindAddr)
// 	fmt.Printf("indexPath: %s\n", *indexPath)
// 	fmt.Printf("cpuprofile: %s\n", *cpuprofile)
// 	fmt.Printf("memprofile: %s\n", *memprofile)
// }

// func main() {
// 	flag.Parse()
// 	printConfig()
// 	log.Printf("GOMAXPROCS: %d", runtime.GOMAXPROCS(-1))

// 	if *cpuprofile != "" {
// 		f, err := os.Create(*cpuprofile)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		pprof.StartCPUProfile(f)
// 	}
// 	// open the index
// 	tableIndex, err := bleve.Open(*indexPath)
// 	if err == bleve.ErrorIndexPathDoesNotExist {
// 		log.Printf("Creating new index...")
// 		//create a mapping
// 		indexMapping, err := buildIndexMapping()
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		tableIndex, err = bleve.New(*indexPath, indexMapping)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		//index data in the background
// 		go func() {
// 			err = indexTables(tableIndex)
// 			if err != nil {
// 				log.Fatal(err)
// 			}
// 			pprof.StopCPUProfile()
// 			if *memprofile != "" {
// 				f, err := os.Create(*memprofile)
// 				if err != nil {
// 					log.Fatal(err)
// 				}
// 				pprof.WriteHeapProfile(f)
// 				f.Close()
// 			}
// 		}()
// 	} else if err != nil {
// 		log.Fatal(err)
// 	} else {
// 		log.Printf("Opening existing index...")
// 	}
// 	// Try a query:
// 	log.Printf("running example search for 'NFL'")
// 	exampleQueryString := "NFL"
// 	query := bleve.NewQueryStringQuery(exampleQueryString)
// 	search := bleve.NewSearchRequest(query)
// 	searchResults, err := tableIndex.Search(search)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Println(searchResults)

// 	// ***ignore/ web server not used
// 	// router := mux.NewRouter()
// 	// bleveHttp.RegisterIndexName("table", tableIndex)
// 	// searchHandler := bleveHttp.NewSearchHandler("table")
// 	// router.Handle("/api/search", searchHandler).Methods("POST")
// 	// // start the HTTP server
// 	// http.Handle("/", router)
// 	// log.Printf("Listening on %v", *bindAddr)
// 	// log.Fatal(http.ListenAndServe(*bindAddr, nil))
// 	// web server not used
// }
