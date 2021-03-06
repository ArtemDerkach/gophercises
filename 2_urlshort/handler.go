package urlshort

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/boltdb/bolt"
	"gopkg.in/yaml.v2"
)

type UrlMap struct {
	Path string `yaml:"path"`
	Url  string `yaml:"url"`
}

// MapHandler will return an http.HandlerFunc (which also
// implements http.Handler) that will attempt to map any
// paths (keys in the map) to their corresponding URL (values
// that each key in the map points to, in string format).
// If the path is not provided in the map, then the fallback
// http.Handler will be called instead.
func MapHandler(pathsToUrls map[string]string, fallback http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("incoming request:", r.URL.Path)
		for path, url := range pathsToUrls {
			fmt.Println(path, url)
			if r.URL.Path != path {
				continue

			}
			http.Redirect(w, r, url, http.StatusMovedPermanently)
			return
		}
		fallback.ServeHTTP(w, r)
	}
}

// YAMLHandler will parse the provided YAML and then return
// an http.HandlerFunc (which also implements http.Handler)
// that will attempt to map any paths to their corresponding
// URL. If the path is not provided in the YAML, then the
// fallback http.Handler will be called instead.
//
// YAML is expected to be in the format:
//
//     - path: /some-path
//       url: https://www.some-url.com/demo
//
// The only errors that can be returned all related to having
// invalid YAML data.
//
// See MapHandler to create a similar http.HandlerFunc via
// a mapping of paths to urls.
func YAMLHandler(yml []byte, fallback http.Handler) (http.HandlerFunc, error) {
	var parsedYaml []UrlMap
	err := yaml.Unmarshal(yml, &parsedYaml)
	if err != nil {
		return nil, err
	}
	pathMap := buildMap(parsedYaml)
	return MapHandler(pathMap, fallback), nil
}

func JSONHandler(jsn []byte, fallback http.Handler) (http.HandlerFunc, error) {
	var parsedJson []UrlMap
	err := json.Unmarshal(jsn, &parsedJson)
	if err != nil {
		return nil, err
	}
	fmt.Println("parsedJson", parsedJson)
	pathMap := buildMap(parsedJson)
	return MapHandler(pathMap, fallback), nil
}

func buildMap(parsedYaml []UrlMap) map[string]string {
	urlPathMap := make(map[string]string)
	for _, urlMap := range parsedYaml {
		urlPathMap[urlMap.Path] = urlMap.Url
	}
	return urlPathMap
}

func DBHandler(db *bolt.DB, fallback http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("urls"))
			c := b.Cursor()

			for k, v := c.First(); k != nil; k, v = c.Next() {
				fmt.Printf("key=%s, value=%s\n", k, v)
				if string(k) != r.URL.Path {
					continue
				}
				http.Redirect(w, r, string(v), http.StatusMovedPermanently)
				return nil
			}

			return nil
		})
		if err != nil {
			w.Write([]byte(err.Error()))
		}
	}
}
